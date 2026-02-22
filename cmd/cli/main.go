package main

import (
	"log"
	"time"

	"github.com/bitcask-iot/engine/client"
)

func main() {
	addr := "127.0.0.1:8080"
	log.Printf("🔌 准备连接服务端: %s", addr)

	// 1. 拨号连接 Server
	c, err := client.NewClient(addr)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer c.Close()
	log.Printf("✅ 连接成功！")

	// 2. 测试发送 PUT 请求 (写数据)
	key := []byte("sensor_temp")
	val := []byte("25.5")
	log.Printf("-> 正在发送 PUT 请求 [Key: %s, Value: %s]", string(key), string(val))

	if err := c.Put(key, val); err != nil {
		log.Printf("❌ PUT 失败: %v", err)
	} else {
		log.Printf("✅ PUT 成功！(服务端回复了 OK)")
	}

	time.Sleep(1 * time.Second) // 稍微停顿1秒，让你能在终端看清日志的先后顺序

	// 3. 测试发送 GET 请求 (读数据)
	log.Printf("-> 正在发送 GET 请求 [Key: %s]", string(key))

	resp, err := c.Get(key)
	if err != nil {
		log.Printf("❌ GET 失败: %v", err)
	} else {
		// 注意：目前我们的 Server 还是个“回声筒”，不管 GET 什么都会回 "OK"
		log.Printf("✅ GET 成功！收到返回值: %s", string(resp))
	}
}
