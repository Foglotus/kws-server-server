package asr

import "testing"

func TestNewASRTaskGeneratesSecureID(t *testing.T) {
	const totalTasks = 256
	seen := make(map[string]struct{}, totalTasks)

	for i := 0; i < totalTasks; i++ {
		task := NewASRTask([]float32{0.1}, 16000, nil, false)
		if !IsValidTaskID(task.ID) {
			t.Fatalf("generated task ID has invalid format: %s", task.ID)
		}
		if _, ok := seen[task.ID]; ok {
			t.Fatalf("duplicate task ID generated: %s", task.ID)
		}
		seen[task.ID] = struct{}{}

		// 释放 context 计时器，避免测试中累积等待超时的任务。
		task.cancel()
	}
}

func TestIsValidTaskID(t *testing.T) {
	validID := generateTaskID()
	if !IsValidTaskID(validID) {
		t.Fatalf("expected generated task ID to be valid: %s", validID)
	}

	invalidIDs := []string{
		"",
		"task_",
		"task_123",
		"task-1234567890abcdef1234567890abcdef",
		"TASK_1234567890abcdef1234567890abcdef",
		"task_1234567890abcdef1234567890abcde",
		"task_1234567890abcdef1234567890abcdef0",
		"task_1234567890abcdef1234567890abcdeg",
	}

	for _, id := range invalidIDs {
		if IsValidTaskID(id) {
			t.Fatalf("expected invalid task ID: %s", id)
		}
	}
}
