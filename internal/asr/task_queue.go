package asr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"airecorder/internal/config"
)

const (
	taskIDPrefix      = "task_"
	taskIDRandomBytes = 16
	taskIDHexLength   = taskIDRandomBytes * 2
)

var taskIDFallbackCounter uint64

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusProcessing
	TaskStatusCompleted
	TaskStatusFailed
)

// ASRTask 语音识别任务
type ASRTask struct {
	ID             string
	Samples        []float32
	SampleRate     int
	DiarizationMgr *DiarizationManager
	EnableDiar     bool
	Result         *ASRTaskResult
	Status         TaskStatus
	SubmitTime     time.Time
	StartTime      time.Time
	CompleteTime   time.Time
	Error          error
	TotalChunks    int32 // 总分块数（原子操作）
	DoneChunks     int32 // 已完成分块数（原子操作）
	ctx            context.Context
	cancel         context.CancelFunc
	resultChan     chan *ASRTaskResult
	mu             sync.RWMutex
}

// ASRTaskResult 任务结果
type ASRTaskResult struct {
	Text     string
	Segments []DiarizationSegment
	Duration float32
	Error    error
}

// Cancel 取消任务
func (t *ASRTask) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status == TaskStatusPending || t.Status == TaskStatusProcessing {
		t.Status = TaskStatusFailed
		t.Error = fmt.Errorf("task cancelled by admin")
		t.CompleteTime = time.Now()
		t.cancel()
		select {
		case t.resultChan <- &ASRTaskResult{Error: t.Error}:
		default:
		}
	}
}

// NewASRTask 创建新任务
func NewASRTask(samples []float32, sampleRate int, diarizationMgr *DiarizationManager, enableDiar bool) *ASRTask {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	submitTime := time.Now()
	return &ASRTask{
		ID:             generateTaskID(),
		Samples:        samples,
		SampleRate:     sampleRate,
		DiarizationMgr: diarizationMgr,
		EnableDiar:     enableDiar,
		Status:         TaskStatusPending,
		SubmitTime:     submitTime,
		ctx:            ctx,
		cancel:         cancel,
		resultChan:     make(chan *ASRTaskResult, 1),
	}
}

func generateTaskID() string {
	entropy := make([]byte, taskIDRandomBytes)
	if _, err := rand.Read(entropy); err != nil {
		fallbackSeed := fmt.Sprintf("%d_%d", time.Now().UnixNano(), atomic.AddUint64(&taskIDFallbackCounter, 1))
		digest := sha256.Sum256([]byte(fallbackSeed))
		log.Printf("[TaskQueue] WARN: crypto/rand unavailable for task ID generation: %v", err)
		return taskIDPrefix + hex.EncodeToString(digest[:taskIDRandomBytes])
	}

	return taskIDPrefix + hex.EncodeToString(entropy)
}

// IsValidTaskID 校验 taskId 格式，避免异常输入探测
func IsValidTaskID(id string) bool {
	if len(id) != len(taskIDPrefix)+taskIDHexLength {
		return false
	}
	if !strings.HasPrefix(id, taskIDPrefix) {
		return false
	}

	for _, ch := range id[len(taskIDPrefix):] {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}

	return true
}

// Wait 等待任务完成
func (t *ASRTask) Wait() *ASRTaskResult {
	select {
	case result := <-t.resultChan:
		return result
	case <-t.ctx.Done():
		return &ASRTaskResult{
			Error: fmt.Errorf("task timeout"),
		}
	}
}

// Complete 标记任务完成
func (t *ASRTask) Complete(result *ASRTaskResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Result = result
	if result.Error != nil {
		t.Status = TaskStatusFailed
		t.Error = result.Error
	} else {
		t.Status = TaskStatusCompleted
	}
	t.CompleteTime = time.Now()

	select {
	case t.resultChan <- result:
	default:
	}

	t.cancel()
}

// GetProgress 获取处理进度百分比 (0-100)
func (t *ASRTask) GetProgress() float32 {
	status := t.GetStatus()
	if status == TaskStatusCompleted {
		return 100
	}
	total := atomic.LoadInt32(&t.TotalChunks)
	if total <= 0 {
		return 0
	}
	done := atomic.LoadInt32(&t.DoneChunks)
	return float32(done) / float32(total) * 100
}

// GetStatus 获取任务状态
func (t *ASRTask) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// SetStatus 设置任务状态
func (t *ASRTask) SetStatus(status TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
	if status == TaskStatusProcessing {
		t.StartTime = time.Now()
	}
}

// TaskQueue 任务队列管理器
type TaskQueue struct {
	config       *config.Config
	asrManager   *OfflineASRManager
	queue        chan *ASRTask
	maxWorkers   int
	maxQueueSize int
	workers      []*worker
	wg           sync.WaitGroup
	shutdown     chan struct{}
	stats        taskQueueStats
	mu           sync.RWMutex
	taskStore    map[string]*ASRTask // taskId -> task 的内存存储
	storeMu      sync.RWMutex
}

type taskQueueStats struct {
	totalTasks      int64
	completedTasks  int64
	failedTasks     int64
	queuedTasks     int64
	processingTasks int64
	totalWaitTime   int64 // 毫秒
	totalExecTime   int64 // 毫秒
}

type worker struct {
	id          int
	queue       *TaskQueue
	processing  atomic.Bool
	currentTask *ASRTask
	mu          sync.RWMutex
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(cfg *config.Config, asrManager *OfflineASRManager) *TaskQueue {
	maxWorkers := 2 // 默认2个worker，避免资源占用过高
	if cfg.Concurrency.WorkerPoolSize > 0 {
		maxWorkers = cfg.Concurrency.WorkerPoolSize
	}

	maxQueueSize := 100 // 默认队列大小
	if cfg.Concurrency.QueueSize > 0 {
		maxQueueSize = cfg.Concurrency.QueueSize
	}

	tq := &TaskQueue{
		config:       cfg,
		asrManager:   asrManager,
		queue:        make(chan *ASRTask, maxQueueSize),
		maxWorkers:   maxWorkers,
		maxQueueSize: maxQueueSize,
		workers:      make([]*worker, maxWorkers),
		shutdown:     make(chan struct{}),
		taskStore:    make(map[string]*ASRTask),
	}

	// 启动worker
	for i := 0; i < maxWorkers; i++ {
		w := &worker{
			id:    i,
			queue: tq,
		}
		tq.workers[i] = w
		tq.wg.Add(1)
		go w.run()
	}

	// 启动任务清理协程（1小时后清理已完成/失败任务）
	go tq.cleanupLoop()

	log.Printf("TaskQueue initialized: max_workers=%d, queue_size=%d", maxWorkers, maxQueueSize)
	return tq
}

// StoreTask 将任务保存到内存存储
func (tq *TaskQueue) StoreTask(task *ASRTask) {
	tq.storeMu.Lock()
	defer tq.storeMu.Unlock()
	tq.taskStore[task.ID] = task
}

// GetTask 根据 ID 从内存存储中查询任务
func (tq *TaskQueue) GetTask(id string) (*ASRTask, bool) {
	tq.storeMu.RLock()
	defer tq.storeMu.RUnlock()
	task, ok := tq.taskStore[id]
	return task, ok
}

// cleanupLoop 定期清理超过1小时的已完成/失败任务
func (tq *TaskQueue) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-tq.shutdown:
			return
		case <-ticker.C:
			tq.storeMu.Lock()
			deadline := time.Now().Add(-1 * time.Hour)
			for id, task := range tq.taskStore {
				status := task.GetStatus()
				if (status == TaskStatusCompleted || status == TaskStatusFailed) &&
					task.CompleteTime.Before(deadline) {
					delete(tq.taskStore, id)
				}
			}
			tq.storeMu.Unlock()
		}
	}
}

// Submit 提交任务
func (tq *TaskQueue) Submit(task *ASRTask) error {
	atomic.AddInt64(&tq.stats.totalTasks, 1)
	atomic.AddInt64(&tq.stats.queuedTasks, 1)

	select {
	case tq.queue <- task:
		log.Printf("[TaskQueue] Task %s submitted, queue_length=%d", task.ID, len(tq.queue))
		return nil
	case <-time.After(5 * time.Second):
		atomic.AddInt64(&tq.stats.queuedTasks, -1)
		return fmt.Errorf("queue is full, please try again later")
	}
}

// GetStats 获取统计信息
func (tq *TaskQueue) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_tasks":      atomic.LoadInt64(&tq.stats.totalTasks),
		"completed_tasks":  atomic.LoadInt64(&tq.stats.completedTasks),
		"failed_tasks":     atomic.LoadInt64(&tq.stats.failedTasks),
		"queued_tasks":     atomic.LoadInt64(&tq.stats.queuedTasks),
		"processing_tasks": atomic.LoadInt64(&tq.stats.processingTasks),
		"queue_length":     len(tq.queue),
		"max_workers":      tq.maxWorkers,
		"max_queue_size":   tq.maxQueueSize,
		"avg_wait_time_ms": tq.getAvgWaitTime(),
		"avg_exec_time_ms": tq.getAvgExecTime(),
	}
}

func (tq *TaskQueue) getAvgWaitTime() int64 {
	completed := atomic.LoadInt64(&tq.stats.completedTasks)
	if completed == 0 {
		return 0
	}
	return atomic.LoadInt64(&tq.stats.totalWaitTime) / completed
}

func (tq *TaskQueue) getAvgExecTime() int64 {
	completed := atomic.LoadInt64(&tq.stats.completedTasks)
	if completed == 0 {
		return 0
	}
	return atomic.LoadInt64(&tq.stats.totalExecTime) / completed
}

// Close 关闭队列
func (tq *TaskQueue) Close() {
	log.Println("Closing TaskQueue...")
	close(tq.shutdown)
	close(tq.queue)
	tq.wg.Wait()
	log.Println("TaskQueue closed")
}

// ListTasks 返回所有任务的摘要信息
func (tq *TaskQueue) ListTasks() []map[string]interface{} {
	tq.storeMu.RLock()
	defer tq.storeMu.RUnlock()

	tasks := make([]map[string]interface{}, 0, len(tq.taskStore))
	for _, task := range tq.taskStore {
		statusStr := "unknown"
		switch task.GetStatus() {
		case TaskStatusPending:
			statusStr = "pending"
		case TaskStatusProcessing:
			statusStr = "processing"
		case TaskStatusCompleted:
			statusStr = "completed"
		case TaskStatusFailed:
			statusStr = "failed"
		}

		var startTimeStr, completeTimeStr string
		if !task.StartTime.IsZero() {
			startTimeStr = task.StartTime.Format(time.RFC3339)
		}
		if !task.CompleteTime.IsZero() {
			completeTimeStr = task.CompleteTime.Format(time.RFC3339)
		}

		var audioSec float32
		if task.SampleRate > 0 {
			audioSec = float32(len(task.Samples)) / float32(task.SampleRate)
		}

		tasks = append(tasks, map[string]interface{}{
			"id":                 task.ID,
			"status":             statusStr,
			"submit_time":        task.SubmitTime.Format(time.RFC3339),
			"start_time":         startTimeStr,
			"complete_time":      completeTimeStr,
			"progress":           task.GetProgress(),
			"audio_duration_sec": audioSec,
			"enable_diarization": task.EnableDiar,
		})
	}
	return tasks
}

// CancelTask 取消指定任务
func (tq *TaskQueue) CancelTask(id string) error {
	task, ok := tq.GetTask(id)
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}
	task.Cancel()
	log.Printf("[TaskQueue] Task %s cancelled by admin", id)
	return nil
}

// GetWorkerStatus 返回 worker 状态列表
func (tq *TaskQueue) GetWorkerStatus() []map[string]interface{} {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(tq.workers))
	for _, w := range tq.workers {
		info := map[string]interface{}{
			"id":           w.id,
			"processing":   w.processing.Load(),
			"current_task": nil,
		}
		w.mu.RLock()
		if w.currentTask != nil {
			info["current_task"] = w.currentTask.ID
		}
		w.mu.RUnlock()
		result = append(result, info)
	}
	return result
}

// worker运行逻辑
func (w *worker) run() {
	defer w.queue.wg.Done()
	log.Printf("[Worker %d] Started", w.id)

	for {
		select {
		case <-w.queue.shutdown:
			log.Printf("[Worker %d] Shutting down", w.id)
			return
		case task, ok := <-w.queue.queue:
			if !ok {
				log.Printf("[Worker %d] Queue closed", w.id)
				return
			}
			w.processTask(task)
		}
	}
}

func (w *worker) processTask(task *ASRTask) {
	w.processing.Store(true)
	w.mu.Lock()
	w.currentTask = task
	w.mu.Unlock()
	defer func() {
		w.processing.Store(false)
		w.mu.Lock()
		w.currentTask = nil
		w.mu.Unlock()
	}()

	atomic.AddInt64(&w.queue.stats.queuedTasks, -1)
	atomic.AddInt64(&w.queue.stats.processingTasks, 1)
	defer atomic.AddInt64(&w.queue.stats.processingTasks, -1)

	// 更新任务状态
	task.SetStatus(TaskStatusProcessing)

	// 记录等待时间
	waitTime := time.Since(task.SubmitTime).Milliseconds()
	atomic.AddInt64(&w.queue.stats.totalWaitTime, waitTime)

	audioDuration := float32(len(task.Samples)) / float32(task.SampleRate)
	log.Printf("[Worker %d] Processing task %s: duration=%.2fs, wait_time=%dms",
		w.id, task.ID, audioDuration, waitTime)

	startTime := time.Now()

	// 执行识别
	var result ASRTaskResult
	result.Duration = audioDuration

	progCb := func(total, completed int) {
		atomic.StoreInt32(&task.TotalChunks, int32(total))
		atomic.StoreInt32(&task.DoneChunks, int32(completed))
	}

	if task.EnableDiar && task.DiarizationMgr != nil {
		// 带说话者分离
		segments, err := task.DiarizationMgr.ProcessWithASR(task.Samples, task.SampleRate, w.queue.asrManager, progCb)
		if err != nil {
			result.Error = err
		} else {
			fullText := ""
			for _, seg := range segments {
				fullText += seg.Text + " "
			}
			result.Text = fullText
			result.Segments = segments
		}
	} else {
		// 普通识别
		chunkDurationSec := w.queue.asrManager.GetChunkDurationSec()
		var text string
		var err error

		if audioDuration > float32(chunkDurationSec) {
			log.Printf("[Worker %d] Using chunked processing for task %s", w.id, task.ID)
			text, err = w.queue.asrManager.RecognizeChunked(task.Samples, task.SampleRate, progCb)
		} else {
			text, err = w.queue.asrManager.Recognize(task.Samples, task.SampleRate)
		}

		result.Text = text
		result.Error = err
	}

	// 记录执行时间
	execTime := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&w.queue.stats.totalExecTime, execTime)

	// 完成任务
	task.Complete(&result)

	if result.Error != nil {
		atomic.AddInt64(&w.queue.stats.failedTasks, 1)
		log.Printf("[Worker %d] Task %s failed: error=%v, exec_time=%dms",
			w.id, task.ID, result.Error, execTime)
	} else {
		atomic.AddInt64(&w.queue.stats.completedTasks, 1)
		log.Printf("[Worker %d] Task %s completed: result_length=%d chars, exec_time=%dms",
			w.id, task.ID, len(result.Text), execTime)
	}
}
