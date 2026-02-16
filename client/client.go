package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/bitcask-iot/engine/protocol"
)

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrTimeout          = errors.New("operation timeout")
)

// Client 客户端 SDK
type Client struct {
	address string
	conn    net.Conn
	reader  *bufio.Reader
	mu      sync.Mutex
}

// NewClient 创建客户端
func NewClient(address string) *Client {
	return &Client{
		address: address,
	}
}

// Connect 连接到服务器
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", c.address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s failed: %w", c.address, err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	return nil
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.reader = nil
		return err
	}

	return nil
}

// isConnected 检查是否已连接
func (c *Client) isConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Write 写入数据点
func (c *Client) Write(sensorID string, timestamp int64, value float64) error {
	if !c.isConnected() {
		return errors.New("not connected to server")
	}

	// 构造写入请求
	req := &protocol.WriteRequest{
		SensorID:  sensorID,
		Timestamp: timestamp,
		Value:     value,
	}

	// 编码请求
	msg, err := protocol.EncodeWriteRequest(req)
	if err != nil {
		return fmt.Errorf("encode write request failed: %w", err)
	}

	// 发送请求并获取响应
	respMsg, err := c.sendAndReceive(msg)
	if err != nil {
		return err
	}

	// 解码响应
	resp, err := protocol.DecodeWriteResponse(respMsg)
	if err != nil {
		return fmt.Errorf("decode write response failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("write failed: %s", resp.Message)
	}

	return nil
}

// Query 查询数据点
func (c *Client) Query(sensorID string, start, end int64) ([]protocol.Point, error) {
	if !c.isConnected() {
		return nil, errors.New("not connected to server")
	}

	// 构造查询请求
	req := &protocol.QueryRequest{
		SensorID: sensorID,
		Start:    start,
		End:      end,
	}

	// 编码请求
	msg, err := protocol.EncodeQueryRequest(req)
	if err != nil {
		return nil, fmt.Errorf("encode query request failed: %w", err)
	}

	// 发送请求并获取响应
	respMsg, err := c.sendAndReceive(msg)
	if err != nil {
		return nil, err
	}

	// 解码响应
	resp, err := protocol.DecodeQueryResponse(respMsg)
	if err != nil {
		return nil, fmt.Errorf("decode query response failed: %w", err)
	}

	return resp.Points, nil
}

// sendAndReceive 发送请求并接收响应
func (c *Client) sendAndReceive(msg []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, errors.New("not connected to server")
	}

	// 设置写超时
	c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// 发送请求
	if _, err := c.conn.Write(msg); err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}

	// 读取响应头部
	headerBuf := make([]byte, 10)
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, err := io.ReadFull(c.reader, headerBuf)
	if err != nil {
		return nil, fmt.Errorf("read response header failed: %w", err)
	}

	// 解析头部，获取 payload 长度
	payloadLen, err := protocol.GetPayloadLength(headerBuf)
	if err != nil {
		return nil, fmt.Errorf("parse response header failed: %w", err)
	}

	// 读取 payload
	payloadBuf := make([]byte, payloadLen)
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, err = io.ReadFull(c.reader, payloadBuf)
	if err != nil {
		return nil, fmt.Errorf("read response payload failed: %w", err)
	}

	// 组合完整响应
	resp := append(headerBuf, payloadBuf...)
	return resp, nil
}

// WriteBatch 批量写入数据点
func (c *Client) WriteBatch(sensorID string, points []protocol.Point) error {
	for _, p := range points {
		if err := c.Write(sensorID, p.Timestamp, p.Value); err != nil {
			return err
		}
	}
	return nil
}

// Ping 检测连接是否可用
func (c *Client) Ping() error {
	if !c.isConnected() {
		return errors.New("not connected to server")
	}

	// 使用一个简单的写入来检测连接
	testSensorID := "__ping__"
	testTimestamp := time.Now().Unix()
	testValue := 0.0

	return c.Write(testSensorID, testTimestamp, testValue)
}