package tcp

import (
	"log"
	"net"
)

// StartServer 启动 TCP 服务端大门
// 目前 MVP 阶段先不传入 db，专心搞网络联调
func StartServer(addr string) error {
	// 1. 申请一个 TCP 端口作为“门面”
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("🚀 迎宾大厅已开启，正在监听端口: %s", addr)

	// 2. 迎宾员进入死循环，等待客人敲门
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接收连接失败: %v", err)
			continue
		}

		// 3. 极其关键：客人来了，立刻派一个专属服务员 (Goroutine) 去接待他
		// 这样迎宾员就能瞬间回到门口等下一个客人，不会阻塞！
		go HandleConnection(conn)
	}
}
