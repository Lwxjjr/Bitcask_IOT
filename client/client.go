package client

import (
	"fmt"
	"net"

	"github.com/bitcask-iot/engine/protocol"
)

// Client 是我们与服务端通信的 SDK 载体
type Client struct {
	conn net.Conn
}

// NewClient 拨号连接服务端
func NewClient(addr string) (*Client, error) {
	// net.Dial 就像是打电话，尝试连接服务端的 IP 和端口
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("连接服务端失败: %v", err)
	}
	return &Client{conn: conn}, nil
}

// Close 断开连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Put 发送写入请求
func (c *Client) Put(key, value []byte) error {
	// 1. 组装请求包
	req := &protocol.Packet{
		Type:  protocol.TypePut,
		Key:   key,
		Value: value,
	}

	// 2. 封包并发送
	if _, err := c.conn.Write(protocol.Encode(req)); err != nil {
		return fmt.Errorf("发送 Put 请求失败: %v", err)
	}

	// 3. 阻塞等待并拆解服务端的回复
	resp, err := protocol.Decode(c.conn)
	if err != nil {
		return fmt.Errorf("读取服务端回复失败: %v", err)
	}

	// 4. 检查服务端是否报错
	if resp.Type == protocol.TypeError {
		return fmt.Errorf("服务端报错: %s", string(resp.Value))
	}

	return nil
}

// Get 发送读取请求
func (c *Client) Get(key []byte) ([]byte, error) {
	// 1. 组装请求包 (Get 请求不需要传 Value)
	req := &protocol.Packet{
		Type: protocol.TypeGet,
		Key:  key,
	}

	// 2. 封包并发送
	if _, err := c.conn.Write(protocol.Encode(req)); err != nil {
		return nil, fmt.Errorf("发送 Get 请求失败: %v", err)
	}

	// 3. 阻塞等待并拆解服务端的回复
	resp, err := protocol.Decode(c.conn)
	if err != nil {
		return nil, fmt.Errorf("读取服务端回复失败: %v", err)
	}

	// 4. 检查服务端是否报错
	if resp.Type == protocol.TypeError {
		return nil, fmt.Errorf("服务端报错: %s", string(resp.Value))
	}

	// 5. 返回服务端查询到的 Value
	return resp.Value, nil
}
