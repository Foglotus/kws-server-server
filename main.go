package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"airecorder/internal/config"
	"airecorder/internal/server"
	"airecorder/internal/version"
)

var (
	showVersion = flag.Bool("version", false, "显示版本信息")
	showV       = flag.Bool("v", false, "显示版本信息（简短）")
)

func main() {
	flag.Parse()

	// 处理版本信息显示
	if *showVersion {
		info := version.Get()
		fmt.Println(info.String())
		os.Exit(0)
	}

	if *showV {
		fmt.Printf("AI Recorder %s\n", version.Short())
		os.Exit(0)
	}

	// 打印启动信息
	log.Printf("AI Recorder %s starting...", version.Short())

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化服务器
	srv := server.NewServer(cfg)

	// 启动服务器
	log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
