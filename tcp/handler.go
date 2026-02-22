package tcp

import (
	"io"
	"log"
	"net"

	"github.com/bitcask-iot/engine/protocol"
)

// HandleConnection 是每个客户端独享的接待流程
func HandleConnection(conn net.Conn) {
	// 无论发生什么，客人走的时候一定要销毁这根网线，释放资源
	defer func() {
		log.Printf("👋 客户端已断开: %s", conn.RemoteAddr().String())
		conn.Close()
	}()

	log.Printf("🎉 新客户端接入: %s", conn.RemoteAddr().String())

	// 服务员进入死循环，只要客人不断开，就一直等他的命令
	for {
		// 1. 拆快递：调用我们写的极简防粘包神技
		reqPacket, err := protocol.Decode(conn)
		if err != nil {
			if err == io.EOF {
				// EOF (End Of File) 说明客人主动拔网线走了，属于正常断开
				break
			}
			log.Printf("❌ 解码错误: %v", err)
			break // 包格式错了，直接踢掉这个客户端
		}

		// 2. 看看客人发了什么 (MVP 阶段先打印出来)
		log.Printf("收到指令 -> Type: %d, Key: %s, Value: %s",
			reqPacket.Type, string(reqPacket.Key), string(reqPacket.Value))

		// 3. 假装后厨已经处理完了，给客人打包一个回复
		// 构造一个回复包 (Type = 3 表示正常响应，Value = "OK")
		respPacket := &protocol.Packet{
			Type:  protocol.TypeReply,
			Key:   nil, // 回复不需要带 Key 了
			Value: []byte("OK"),
		}

		// 4. 寄快递：打包并塞回网线发给客人
		encodedBytes := protocol.Encode(respPacket)
		_, err = conn.Write(encodedBytes)
		if err != nil {
			log.Printf("❌ 回复客户端失败: %v", err)
			break
		}
	}
}
