package protocol

import (
	"encoding/binary"
	"io"
)

// ==========================================
// 1. 定义 MVP 阶段的核心指令 (Type)
// ==========================================
const (
	// 客户端 -> 服务端
	TypeWrite uint8 = 1 // 写入数据
	TypeQuery uint8 = 2 // 读取数据

	// 服务端 -> 客户端
	TypeReply uint8 = 3 // 正常响应 (带数据或仅代表成功)
	TypeError uint8 = 4 // 错误响应 (Value 里是错误信息)
)

// HeaderSize 包头固定长度: Type(1字节) + KeyLen(4字节) + ValLen(4字节) = 9 bytes
const HeaderSize = 9

// Packet 是我们的标准网络数据包
// 结构: [Type][KeyLen][ValLen][Key内容][Value内容]
type Packet struct {
	Type     uint8
	KeyLen   uint32
	ValueLen uint32
	Key      []byte
	Value    []byte
}

// ==========================================
// 2. 核心方法：封包与拆包
// ==========================================

// Encode 封包：将 Packet 结构体转换为二进制字节流 (发数据用)
func Encode(p *Packet) []byte {
	// 1. 计算总包长
	size := HeaderSize + len(p.Key) + len(p.Value)
	buf := make([]byte, size)

	// 2. 按照大端序 (BigEndian) 写入 9 字节 Header
	buf[0] = p.Type
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(p.Key)))
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(p.Value)))

	// 3. 按顺序写入 Key 和 Value 的具体内容
	offset := HeaderSize
	if len(p.Key) > 0 {
		copy(buf[offset:], p.Key)
		offset += len(p.Key)
	}
	if len(p.Value) > 0 {
		copy(buf[offset:], p.Value)
	}

	return buf
}

// Decode 拆包：从连接中安全地读取并还原出 Packet 结构体 (收数据用)
func Decode(r io.Reader) (*Packet, error) {
	// 1. 死等读取 9 个字节的 Header (解决 TCP 粘包的核心)
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err // 可能是网络断开 (io.EOF)
	}

	// 2. 解析 Header 里的信息
	p := &Packet{
		Type:     header[0],
		KeyLen:   binary.BigEndian.Uint32(header[1:5]),
		ValueLen: binary.BigEndian.Uint32(header[5:9]),
	}

	// 3. 根据解析出的 Key 长度，精准读取 Key 内容
	if p.KeyLen > 0 {
		p.Key = make([]byte, p.KeyLen)
		if _, err := io.ReadFull(r, p.Key); err != nil {
			return nil, err
		}
	}

	// 4. 根据解析出的 Value 长度，精准读取 Value 内容
	if p.ValueLen > 0 {
		p.Value = make([]byte, p.ValueLen)
		if _, err := io.ReadFull(r, p.Value); err != nil {
			return nil, err
		}
	}

	return p, nil
}
