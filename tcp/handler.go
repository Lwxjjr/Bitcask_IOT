package tcp

import (
	"encoding/binary"
	"io"
	"log"
	"math"
	"net"

	"github.com/bitcask-iot/engine/core"
	"github.com/bitcask-iot/engine/protocol"
)

func HandleConnection(conn net.Conn, db *core.DB) {
	defer conn.Close()

	for {
		reqPacket, err := protocol.Decode(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("❌ 解码错误: %v", err)
			}
			break
		}

		var respPacket *protocol.Packet
		sensorID := string(reqPacket.Key)

		switch reqPacket.Type {

		case protocol.TypeWrite: // 🌟 捕获写入指令
			if len(reqPacket.Value) == 16 {
				ts := int64(binary.BigEndian.Uint64(reqPacket.Value[0:8]))
				val := math.Float64frombits(binary.BigEndian.Uint64(reqPacket.Value[8:16]))

				// ⚡️ 真正调用你的 DB 门面！
				if err := db.Write(sensorID, ts, val); err != nil {
					respPacket = &protocol.Packet{Type: protocol.TypeError, Value: []byte(err.Error())}
				} else {
					respPacket = &protocol.Packet{Type: protocol.TypeReply, Value: []byte("OK")}
				}
			} else {
				respPacket = &protocol.Packet{Type: protocol.TypeError, Value: []byte("invalid payload size")}
			}

		case protocol.TypeQuery: // 🌟 捕获查询指令
			if len(reqPacket.Value) == 16 {
				start := int64(binary.BigEndian.Uint64(reqPacket.Value[0:8]))
				end := int64(binary.BigEndian.Uint64(reqPacket.Value[8:16]))

				// ⚡️ 调用 DB 进行时间范围查询！
				points, err := db.Query(sensorID, start, end)
				if err != nil {
					respPacket = &protocol.Packet{Type: protocol.TypeError, Value: []byte(err.Error())}
				} else {
					// 将返回的 []Point 压缩成一根连续的二进制水管发回去
					respVal := make([]byte, len(points)*16)
					for i, p := range points {
						binary.BigEndian.PutUint64(respVal[i*16:i*16+8], uint64(p.Time))
						binary.BigEndian.PutUint64(respVal[i*16+8:i*16+16], math.Float64bits(p.Value))
					}
					respPacket = &protocol.Packet{Type: protocol.TypeReply, Value: respVal}
				}
			}

		case protocol.TypeKeys: // 🌟 获取全部 Key
			keys := db.Keys()
			respVal := protocol.EncodeKeys(keys)
			respPacket = &protocol.Packet{Type: protocol.TypeReply, Value: respVal}

		default:
			respPacket = &protocol.Packet{Type: protocol.TypeError, Value: []byte("unknown command")}
		}

		conn.Write(protocol.Encode(respPacket))
	}
}
