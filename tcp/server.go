package tcp

import (
	"log"
	"net"

	"github.com/bitcask-iot/engine/core"
)

// StartServer 启动 TCP 服务端大门 (新增了 db 参数)
func StartServer(addr string, db *core.DB) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("🚀 迎宾大厅已开启，正在监听端口: %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接收连接失败: %v", err)
			continue
		}

		// 🌟 极其关键：把 conn 和 db 一起交给服务员！
		go HandleConnection(conn, db)
	}
}
