package main

import (
	"log"
	"time"

	"github.com/bitcask-iot/engine/client"
)

func main() {
	c, err := client.NewClient("127.0.0.1:8080")
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer c.Close()

	sensorID := "temp_engine_01"

	// 在客户端获取准确的事件时间 (毫秒级)
	now := time.Now().UnixMilli()

	log.Printf("🔌 连接成功！开始时序写入测试...")

	// 1. 连续 Write 3 条数据 (模拟传感器持续上报)
	log.Printf("-> 正在写入 T1: %d, 值: 25.5", now)
	c.Write(sensorID, now, 25.5)

	time.Sleep(100 * time.Millisecond) // 稍微等一下，制造时间差

	now2 := time.Now().UnixMilli()
	log.Printf("-> 正在写入 T2: %d, 值: 26.1", now2)
	c.Write(sensorID, now2, 26.1)

	time.Sleep(100 * time.Millisecond)

	now3 := time.Now().UnixMilli()
	log.Printf("-> 正在写入 T3: %d, 值: 26.8", now3)
	c.Write(sensorID, now3, 26.8)

	log.Printf("✅ 写入完毕！开始测试范围查询...\n")

	// 2. Query 查询刚才这 1 秒内的所有数据
	start := now - 1000 // 往前推 1 秒
	end := now3 + 1000  // 往后推 1 秒

	log.Printf("-> 正在查询范围 [%d] 到 [%d]", start, end)
	points, err := c.Query(sensorID, start, end)
	if err != nil {
		log.Fatalf("❌ Query 失败: %v", err)
	}

	log.Printf("✅ Query 成功！共查出 %d 个点:", len(points))
	for i, p := range points {
		log.Printf("   [%d] 时间戳: %d => 温度: %.2f", i+1, p.Time, p.Value)
	}
}
