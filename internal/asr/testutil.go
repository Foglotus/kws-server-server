package asr

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LoadAudioFile 加载音频文件并转换为 float32 samples
// 支持 wav, mp3, mp4 等格式，使用 ffmpeg 转换
func LoadAudioFile(filePath string, targetSampleRate int) ([]float32, int, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, 0, fmt.Errorf("file not found: %s", filePath)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	// 如果是 WAV 文件，尝试直接读取
	if ext == ".wav" {
		samples, sampleRate, err := loadWavFile(filePath)
		if err == nil {
			// 如果需要重采样
			if targetSampleRate > 0 && sampleRate != targetSampleRate {
				return resampleWithFFmpeg(samples, sampleRate, targetSampleRate)
			}
			return samples, sampleRate, nil
		}
		// 如果直接读取失败，回退到使用 ffmpeg
	}

	// 使用 ffmpeg 转换音频
	return convertWithFFmpeg(filePath, targetSampleRate)
}

// loadWavFile 直接读取 WAV 文件
func loadWavFile(filePath string) ([]float32, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	// 读取 WAV 头部
	header := make([]byte, 44)
	_, err = io.ReadFull(file, header)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV header: %v", err)
	}

	// 验证 WAV 格式
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a valid WAV file")
	}

	// 读取格式信息
	audioFormat := binary.LittleEndian.Uint16(header[20:22])
	numChannels := binary.LittleEndian.Uint16(header[22:24])
	sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := binary.LittleEndian.Uint16(header[34:36])

	// 只支持 PCM 格式
	if audioFormat != 1 {
		return nil, 0, fmt.Errorf("unsupported audio format: %d (only PCM is supported)", audioFormat)
	}

	// 读取音频数据
	audioData, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read audio data: %v", err)
	}

	// 转换为 float32 samples
	var samples []float32
	switch bitsPerSample {
	case 16:
		samples = make([]float32, len(audioData)/2/int(numChannels))
		for i := 0; i < len(samples); i++ {
			// 对于多声道，取第一个声道
			offset := i * int(numChannels) * 2
			if offset+1 < len(audioData) {
				sample := int16(binary.LittleEndian.Uint16(audioData[offset : offset+2]))
				samples[i] = float32(sample) / 32768.0
			}
		}
	case 32:
		samples = make([]float32, len(audioData)/4/int(numChannels))
		for i := 0; i < len(samples); i++ {
			offset := i * int(numChannels) * 4
			if offset+3 < len(audioData) {
				sample := int32(binary.LittleEndian.Uint32(audioData[offset : offset+4]))
				samples[i] = float32(sample) / 2147483648.0
			}
		}
	default:
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
	}

	return samples, sampleRate, nil
}

// convertWithFFmpeg 使用 ffmpeg 转换音频文件
func convertWithFFmpeg(filePath string, targetSampleRate int) ([]float32, int, error) {
	if targetSampleRate == 0 {
		targetSampleRate = 16000 // 默认采样率
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "audio_*.wav")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// 使用 ffmpeg 转换为 16kHz 16-bit PCM WAV
	cmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-ar", fmt.Sprintf("%d", targetSampleRate),
		"-ac", "1", // 单声道
		"-sample_fmt", "s16",
		"-f", "wav",
		"-y",
		tmpPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("ffmpeg conversion failed: %v\nOutput: %s", err, string(output))
	}

	// 读取转换后的文件
	samples, sampleRate, err := loadWavFile(tmpPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load converted file: %v", err)
	}

	return samples, sampleRate, nil
}

// resampleWithFFmpeg 使用 ffmpeg 重采样音频
func resampleWithFFmpeg(samples []float32, fromRate, toRate int) ([]float32, int, error) {
	// 创建临时输入文件
	tmpIn, err := os.CreateTemp("", "audio_in_*.wav")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp input file: %v", err)
	}
	tmpInPath := tmpIn.Name()
	defer os.Remove(tmpInPath)

	// 将 samples 写入 WAV 文件
	err = writeSamplesToWav(tmpIn, samples, fromRate)
	tmpIn.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to write samples to WAV: %v", err)
	}

	// 使用 ffmpeg 重采样
	return convertWithFFmpeg(tmpInPath, toRate)
}

// writeSamplesToWav 将 float32 samples 写入 WAV 文件
func writeSamplesToWav(w io.Writer, samples []float32, sampleRate int) error {
	numSamples := len(samples)
	dataSize := numSamples * 2 // 16-bit samples
	fileSize := 36 + dataSize

	// 写入 WAV 头部
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(header[22:24], 1)  // Mono
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*2)) // Byte rate
	binary.LittleEndian.PutUint16(header[32:34], 2)                    // Block align
	binary.LittleEndian.PutUint16(header[34:36], 16)                   // Bits per sample
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	if _, err := w.Write(header); err != nil {
		return err
	}

	// 写入音频数据
	for _, sample := range samples {
		// 将 float32 转换为 int16
		intSample := int16(sample * 32767.0)
		if err := binary.Write(w, binary.LittleEndian, intSample); err != nil {
			return err
		}
	}

	return nil
}
