package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitcask-iot/engine/core"
	"github.com/bitcask-iot/engine/tcp"
)

func main() {
	// 1. 定义数据存放的物理目录 (改个专业点的名字)
	dataDir := "./tsdb_data"

	log.Printf("📦 正在初始化 TSDB 时序存储引擎, 目录: %s", dataDir)

	// 2. 启动数据库的核心大脑和磁盘管理器
	db, err := core.NewDB(dataDir)
	if err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}

	// 🌟 3. 极其重要的“优雅退出”机制
	// 监听系统的 Ctrl+C (SIGINT) 或 kill (SIGTERM) 信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan // 死等信号，一旦收到就往下执行
		log.Println("\n🛑 收到退出信号，正在安全关闭数据库...")

		// 调起 core.DB 里的 Close()，它会通知后台巡检协程停手，
		// 并且你可以把内存里还没满的 Block 强制刷入磁盘
		if err := db.Close(); err != nil {
			log.Printf("❌ 关闭数据库时发生错误: %v", err)
		} else {
			log.Println("✅ 数据库安全关闭，滞留数据已落盘。")
		}
		os.Exit(0)
	}()

	// 4. 启动网络大门，把建好的 db 实例传给前台服务员
	addr := ":8080"
	if err := tcp.StartServer(addr, db); err != nil {
		log.Fatalf("服务端异常退出: %v", err)
	}
}
