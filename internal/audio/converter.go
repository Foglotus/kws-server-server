package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

// AudioFormat 音频格式类型
type AudioFormat string

const (
	FormatWAV  AudioFormat = "wav"
	FormatMP3  AudioFormat = "mp3"
	FormatM4A  AudioFormat = "m4a"
	FormatFLAC AudioFormat = "flac"
	FormatOGG  AudioFormat = "ogg"
	FormatAAC  AudioFormat = "aac"
	FormatWMA  AudioFormat = "wma"
	FormatAMR  AudioFormat = "amr"
	FormatOPUS AudioFormat = "opus"
)

// AudioConverter 音频转换器
type AudioConverter struct {
	targetSampleRate int
	targetChannels   int
}

// NewAudioConverter 创建音频转换器
func NewAudioConverter() *AudioConverter {
	return &AudioConverter{
		targetSampleRate: 16000, // 目标采样率 16kHz
		targetChannels:   1,     // 目标单声道
	}
}

// ConvertToSamples 将各种音频格式转换为 float32 样本
func (c *AudioConverter) ConvertToSamples(audioData []byte) ([]float32, int, error) {
	// 检测音频格式
	format, err := c.detectFormat(audioData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to detect audio format: %w", err)
	}

	log.Printf("Detected audio format: %s", format)

	// 根据格式进行转换
	switch format {
	case FormatWAV:
		return c.convertWAV(audioData)
	default:
		// 对于非WAV格式，使用FFmpeg进行转换
		return c.convertWithFFmpeg(audioData, format)
	}
}

// detectFormat 检测音频格式
func (c *AudioConverter) detectFormat(data []byte) (AudioFormat, error) {
	if len(data) < 12 {
		return "", fmt.Errorf("audio data too short")
	}

	// 检查 WAV 格式 (RIFF....WAVE)
	if bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE")) {
		return FormatWAV, nil
	}

	// 检查 MP3 格式 (ID3 或 0xFF 0xFB/0xFA)
	if bytes.Equal(data[0:3], []byte("ID3")) || (data[0] == 0xFF && (data[1]&0xE0) == 0xE0) {
		return FormatMP3, nil
	}

	// 检查 M4A/MP4 格式 (ftyp)
	if len(data) >= 8 && bytes.Equal(data[4:8], []byte("ftyp")) {
		return FormatM4A, nil
	}

	// 检查 FLAC 格式
	if bytes.Equal(data[0:4], []byte("fLaC")) {
		return FormatFLAC, nil
	}

	// 检查 OGG 格式
	if bytes.Equal(data[0:4], []byte("OggS")) {
		// 进一步检查是否是Opus
		if len(data) >= 36 && bytes.Equal(data[28:36], []byte("OpusHead")) {
			return FormatOPUS, nil
		}
		return FormatOGG, nil
	}

	// 检查 AMR 格式
	if bytes.Equal(data[0:6], []byte("#!AMR\n")) {
		return FormatAMR, nil
	}

	return "", fmt.Errorf("unsupported audio format")
}

// convertWAV 转换 WAV 格式
func (c *AudioConverter) convertWAV(data []byte) ([]float32, int, error) {
	if len(data) < 44 {
		return nil, 0, fmt.Errorf("invalid WAV file: too short")
	}

	// 解析 WAV 头
	reader := bytes.NewReader(data)

	// 跳过 "RIFF" 和文件大小
	reader.Seek(8, io.SeekStart)

	// 检查 "WAVE"
	wave := make([]byte, 4)
	reader.Read(wave)
	if !bytes.Equal(wave, []byte("WAVE")) {
		return nil, 0, fmt.Errorf("invalid WAV file: missing WAVE marker")
	}

	// 查找 "fmt " chunk
	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var bitsPerSample uint16

	for {
		chunkID := make([]byte, 4)
		if _, err := reader.Read(chunkID); err != nil {
			return nil, 0, fmt.Errorf("failed to read chunk ID: %w", err)
		}

		var chunkSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
			return nil, 0, fmt.Errorf("failed to read chunk size: %w", err)
		}

		if bytes.Equal(chunkID, []byte("fmt ")) {
			binary.Read(reader, binary.LittleEndian, &audioFormat)
			binary.Read(reader, binary.LittleEndian, &numChannels)
			binary.Read(reader, binary.LittleEndian, &sampleRate)
			reader.Seek(6, io.SeekCurrent) // 跳过 ByteRate 和 BlockAlign
			binary.Read(reader, binary.LittleEndian, &bitsPerSample)

			// 跳过剩余的 fmt chunk
			if chunkSize > 16 {
				reader.Seek(int64(chunkSize-16), io.SeekCurrent)
			}
			break
		} else {
			// 跳过此 chunk
			reader.Seek(int64(chunkSize), io.SeekCurrent)
		}
	}

	// 查找 "data" chunk
	var audioData []byte
	for {
		chunkID := make([]byte, 4)
		if _, err := reader.Read(chunkID); err != nil {
			return nil, 0, fmt.Errorf("failed to find data chunk: %w", err)
		}

		var chunkSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
			return nil, 0, fmt.Errorf("failed to read data chunk size: %w", err)
		}

		if bytes.Equal(chunkID, []byte("data")) {
			audioData = make([]byte, chunkSize)
			reader.Read(audioData)
			break
		} else {
			// 跳过此 chunk
			reader.Seek(int64(chunkSize), io.SeekCurrent)
		}
	}

	log.Printf("WAV info: format=%d, channels=%d, sampleRate=%d, bitsPerSample=%d, dataSize=%d",
		audioFormat, numChannels, sampleRate, bitsPerSample, len(audioData))

	// 转换为 float32 样本
	samples, err := c.decodePCM(audioData, bitsPerSample, numChannels)
	if err != nil {
		return nil, 0, err
	}

	// 如果需要重采样
	if sampleRate != uint32(c.targetSampleRate) {
		samples = c.resample(samples, int(sampleRate), c.targetSampleRate)
	}

	return samples, c.targetSampleRate, nil
}

// decodePCM 解码 PCM 数据
func (c *AudioConverter) decodePCM(data []byte, bitsPerSample uint16, numChannels uint16) ([]float32, error) {
	var samples []float32

	switch bitsPerSample {
	case 16:
		// 16位 PCM
		numSamples := len(data) / 2
		samplesPerChannel := numSamples / int(numChannels)
		samples = make([]float32, samplesPerChannel)

		for i := 0; i < samplesPerChannel; i++ {
			// 只处理第一个声道
			idx := i * int(numChannels) * 2
			low := int16(data[idx])
			high := int16(data[idx+1])
			s16 := (high << 8) | low
			samples[i] = float32(s16) / 32768.0
		}

	case 8:
		// 8位 PCM (无符号)
		samplesPerChannel := len(data) / int(numChannels)
		samples = make([]float32, samplesPerChannel)
		for i := 0; i < samplesPerChannel; i++ {
			idx := i * int(numChannels)
			samples[i] = (float32(data[idx]) - 128.0) / 128.0
		}

	case 32:
		// 32位 PCM
		numSamples := len(data) / 4
		samplesPerChannel := numSamples / int(numChannels)
		samples = make([]float32, samplesPerChannel)

		reader := bytes.NewReader(data)
		for i := 0; i < samplesPerChannel; i++ {
			var s32 int32
			binary.Read(reader, binary.LittleEndian, &s32)
			samples[i] = float32(s32) / 2147483648.0

			// 跳过其他声道
			if numChannels > 1 {
				reader.Seek(int64((numChannels-1)*4), io.SeekCurrent)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
	}

	return samples, nil
}

// convertWithFFmpeg 使用 FFmpeg 转换音频格式
func (c *AudioConverter) convertWithFFmpeg(audioData []byte, format AudioFormat) ([]float32, int, error) {
	// 检查 FFmpeg 是否可用
	if !c.isFFmpegAvailable() {
		return nil, 0, fmt.Errorf("FFmpeg is not available. Please install FFmpeg to support %s format", format)
	}

	log.Printf("Converting %s format using FFmpeg...", format)

	// 使用 FFmpeg 将音频转换为 16kHz, 16bit, mono PCM
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0", // 从标准输入读取
		"-ar", fmt.Sprintf("%d", c.targetSampleRate), // 采样率
		"-ac", "1", // 单声道
		"-f", "s16le", // 16位小端PCM
		"-acodec", "pcm_s16le",
		"pipe:1", // 输出到标准输出
	)

	cmd.Stdin = bytes.NewReader(audioData)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return nil, 0, fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	pcmData := stdout.Bytes()
	log.Printf("Converted to PCM: %d bytes", len(pcmData))

	// 转换 PCM 为 float32
	samples, err := c.decodePCM(pcmData, 16, 1)
	if err != nil {
		return nil, 0, err
	}

	return samples, c.targetSampleRate, nil
}

// isFFmpegAvailable 检查 FFmpeg 是否可用
func (c *AudioConverter) isFFmpegAvailable() bool {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// resample 简单的线性重采样（生产环境建议使用更高质量的重采样算法）
func (c *AudioConverter) resample(samples []float32, fromRate, toRate int) []float32 {
	if fromRate == toRate {
		return samples
	}

	log.Printf("Resampling from %d Hz to %d Hz", fromRate, toRate)

	ratio := float64(fromRate) / float64(toRate)
	newLength := int(float64(len(samples)) / ratio)
	resampled := make([]float32, newLength)

	for i := 0; i < newLength; i++ {
		srcIndex := float64(i) * ratio
		srcIndexInt := int(srcIndex)

		if srcIndexInt >= len(samples)-1 {
			resampled[i] = samples[len(samples)-1]
			continue
		}

		// 线性插值
		fraction := srcIndex - float64(srcIndexInt)
		resampled[i] = samples[srcIndexInt]*(1-float32(fraction)) + samples[srcIndexInt+1]*float32(fraction)
	}

	return resampled
}

// GetSupportedFormats 返回支持的音频格式列表
func GetSupportedFormats() []string {
	formats := []string{"wav", "mp3", "m4a", "mp4", "flac", "ogg", "opus", "aac", "wma", "amr"}

	// 检查 FFmpeg 是否可用
	converter := NewAudioConverter()
	if !converter.isFFmpegAvailable() {
		return []string{"wav"}
	}

	return formats
}

// FormatDescription 返回格式描述信息
func FormatDescription() string {
	formats := GetSupportedFormats()

	var desc strings.Builder
	desc.WriteString("Supported audio formats: ")
	desc.WriteString(strings.Join(formats, ", "))

	if len(formats) == 1 {
		desc.WriteString("\nNote: Install FFmpeg to support more audio formats (MP3, M4A, FLAC, OGG, etc.)")
	}

	return desc.String()
}
