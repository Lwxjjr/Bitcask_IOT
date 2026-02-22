package main

import (
	"log"

	"github.com/bitcask-iot/engine/tcp"
)

func main() {
	addr := ":8080"
	log.Printf("🚀 准备启动 Bitcask-IoT 服务端 MVP...")

	// 启动 TCP Server (这个函数内部是个死循环，会一直阻塞在这里)
	if err := tcp.StartServer(addr); err != nil {
		log.Fatalf("服务端异常退出: %v", err)
	}
}
