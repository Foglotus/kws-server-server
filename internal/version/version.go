package version

import (
	"fmt"
	"runtime"
)

var (
	// Version 应用版本号，由编译时通过 -ldflags 注入
	Version = "dev"
	// GitCommit Git 提交哈希，由编译时注入
	GitCommit = "unknown"
	// BuildTime 构建时间，由编译时注入
	BuildTime = "unknown"
	// GoVersion Go 版本
	GoVersion = runtime.Version()
)

// Info 返回版本信息对象
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get 返回版本信息
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回版本字符串
func (i Info) String() string {
	return fmt.Sprintf("AI Recorder %s (commit: %s, built: %s, %s, %s)",
		i.Version, i.GitCommit, i.BuildTime, i.GoVersion, i.Platform)
}

// Short 返回简短版本字符串
func Short() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}
