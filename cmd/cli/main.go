package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bitcask-iot/engine/client"
	"github.com/chzyer/readline"
)

func main() {
	serverAddr := "127.0.0.1:8080"
	c, err := client.NewClient(serverAddr)
	if err != nil {
		log.Fatalf("❌ 连接服务端失败: %v", err)
	}
	defer c.Close()

	printBanner(serverAddr)

	// 🌟 替换掉原来的 bufio.Scanner
	rl, err := readline.New("Bitcask-IoT > ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		// 阻塞等待用户输入，现在支持上下方向键和历史记录了！
		line, err := rl.Readline()
		if err != nil { // 包含 EOF (Ctrl+D) 或中断 (Ctrl+C)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "put", "write":
			handleWrite(c, parts)
		case "get", "query":
			handleQuery(c, parts)
		case "exit", "quit":
			fmt.Println("👋 Bye!")
			return
		case "help":
			printHelp()
		default:
			fmt.Printf("❌ 未知命令: %s (输入 help 查看帮助)\n", cmd)
		}
	}
}

// ==========================================
// 🎮 具体的命令处理逻辑
// ==========================================

// handleWrite 处理写入: put <key> <value> [timestamp]
func handleWrite(c *client.Client, parts []string) {
	if len(parts) < 3 {
		fmt.Println("❌ 格式错误: put <sensor_id> <value> [timestamp]")
		return
	}

	sensorID := parts[1]

	// 解析 value (float64)
	val, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		fmt.Println("❌ Value 必须是数字")
		return
	}

	// 解析 timestamp (如果有第4个参数就用，没有就用现在)
	var ts int64
	if len(parts) >= 4 {
		t, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			fmt.Println("❌ Timestamp 必须是整数毫秒")
			return
		}
		ts = t
	} else {
		ts = time.Now().UnixMilli()
	}

	// 发送请求
	err = c.Write(sensorID, ts, val)
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Printf("✅ 写入成功! [Key:%s, Time:%d, Val:%.2f]\n", sensorID, ts, val)
	}
}

// handleQuery 处理查询: get <key> (默认查最近5分钟)
// 或者: get <key> <start_ts> <end_ts>
func handleQuery(c *client.Client, parts []string) {
	if len(parts) < 2 {
		fmt.Println("❌ 格式错误: get <sensor_id> [start_ts] [end_ts]")
		return
	}

	sensorID := parts[1]

	var start, end int64

	// 智能判断：用户没传时间，默认查“过去5分钟”到“未来1分钟”
	if len(parts) == 2 {
		now := time.Now().UnixMilli()
		start = now - (5 * 60 * 1000) // 5分钟前
		end = now + (60 * 1000)       // 1分钟后
		fmt.Printf("🔍 未指定时间范围，默认查询最近 5 分钟...\n")
	} else if len(parts) == 4 {
		// 用户指定了 start 和 end
		var err error
		start, err = strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			fmt.Println("❌ Start Time 格式错误")
			return
		}
		end, err = strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			fmt.Println("❌ End Time 格式错误")
			return
		}
	} else {
		fmt.Println("❌ 格式错误: 要么不传时间，要么把 start 和 end 都传上")
		return
	}

	// 发送请求
	points, err := c.Query(sensorID, start, end)
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
		return
	}

	// 打印结果
	fmt.Printf("📊 查询结果 (共 %d 条):\n", len(points))
	fmt.Println("------------------------------------------------")
	fmt.Printf("%-25s | %s\n", "Timestamp (Ms)", "Value")
	fmt.Println("------------------------------------------------")
	if len(points) == 0 {
		fmt.Println("   (无数据)")
	}
	for _, p := range points {
		// 把毫秒转成可读的时间字符串
		tStr := time.UnixMilli(p.Time).Format("15:04:05.000")
		fmt.Printf("%s (%d) | %.2f\n", tStr, p.Time, p.Value)
	}
	fmt.Println("------------------------------------------------")
}

func printBanner(addr string) {
	fmt.Println(`
    ____  _ __                 __    
   / __ )(_) /__________ ____ / /__  
  / __  / / __/ ___/ __ / __ / //_/  
 / /_/ / / /_/ /__/ /_/ (__  / ,<    
/_____/_/\__/\___/\__,_/____/_/|_|   
IOT TSDB CLI v1.0
Connected to ` + addr)
	printHelp()
}

func printHelp() {
	fmt.Println(`
命令帮助:
  1. 写入数据 (自动当前时间):
     put <sensor_id> <value>
     例: put temp_01 26.5

  2. 写入历史数据 (指定时间戳):
     put <sensor_id> <value> <timestamp>
     例: put temp_01 26.5 1709880000000

  3. 查询数据 (默认查最近5分钟):
     get <sensor_id>
     例: get temp_01

  4. 查询指定范围:
     get <sensor_id> <start_ts> <end_ts>

  5. 退出:
     exit / quit
---------------------------------------`)
}
