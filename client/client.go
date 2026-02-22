package client

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"

	"github.com/bitcask-iot/engine/protocol"
)

type Client struct {
	conn net.Conn
}

// Point 定义给外部调用的结构体
type Point struct {
	Time  int64
	Value float64
}

func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("连接服务端失败: %v", err)
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Write 🌟 替换原来的 Put，严格对齐时序语义
func (c *Client) Write(sensorID string, timestamp int64, value float64) error {
	// 把 int64 和 float64 揉进 16 字节的切片里
	valBuf := make([]byte, 16)
	binary.BigEndian.PutUint64(valBuf[0:8], uint64(timestamp))
	binary.BigEndian.PutUint64(valBuf[8:16], math.Float64bits(value))

	req := &protocol.Packet{
		Type:  protocol.TypeWrite, // 对应协议里的 1
		Key:   []byte(sensorID),
		Value: valBuf,
	}

	if _, err := c.conn.Write(protocol.Encode(req)); err != nil {
		return err
	}

	resp, err := protocol.Decode(c.conn)
	if err != nil {
		return err
	}
	if resp.Type == protocol.TypeError {
		return fmt.Errorf("服务端报错: %s", string(resp.Value))
	}

	return nil
}

// Query 🌟 替换原来的 Get，支持时间范围扫描
func (c *Client) Query(sensorID string, start, end int64) ([]Point, error) {
	valBuf := make([]byte, 16)
	binary.BigEndian.PutUint64(valBuf[0:8], uint64(start))
	binary.BigEndian.PutUint64(valBuf[8:16], uint64(end))

	req := &protocol.Packet{
		Type:  protocol.TypeQuery, // 对应协议里的 2
		Key:   []byte(sensorID),
		Value: valBuf,
	}

	if _, err := c.conn.Write(protocol.Encode(req)); err != nil {
		return nil, err
	}

	resp, err := protocol.Decode(c.conn)
	if err != nil {
		return nil, err
	}
	if resp.Type == protocol.TypeError {
		return nil, fmt.Errorf("服务端报错: %s", string(resp.Value))
	}

	// 拆解服务端返回的一大坨二进制，还原成 []Point
	var points []Point
	for i := 0; i < len(resp.Value); i += 16 {
		t := int64(binary.BigEndian.Uint64(resp.Value[i : i+8]))
		v := math.Float64frombits(binary.BigEndian.Uint64(resp.Value[i+8 : i+16]))
		points = append(points, Point{Time: t, Value: v})
	}

	return points, nil
}
