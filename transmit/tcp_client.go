// 1.读取UDP客户端数据转发至TCP服务器
// 2.读取TCP服务器数据转发至UDP客户端

package transmit

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gq-tang/transform/setting"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	STATUS_INIT int = iota
	STATUS_CONNECT
	STATUS_CLOSE
)

// tcp客户端
type TCPClient struct {
	conn           *net.TCPConn
	writeCh        chan []byte
	readCh         chan []byte
	status         int
	readBufferSize int
	updAddr        *net.UDPAddr
	mu             sync.Mutex
}

// 新建TCP客户端，一个UDP客户端对应一个TCP客户端
func NewTCPClient(conn *net.TCPConn, bufSize int, udpAddr *net.UDPAddr) *TCPClient {
	return &TCPClient{
		conn:           conn,
		writeCh:        make(chan []byte, 10),
		readCh:         make(chan []byte, 10),
		status:         STATUS_CONNECT,
		readBufferSize: bufSize,
		updAddr:        udpAddr,
	}
}

// 设置TCP客户端对应的UDP地址
func (c *TCPClient) SetUDPAddr(udpaddr *net.UDPAddr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updAddr = udpaddr
}

// 发送数据至TCP服务器
func (c *TCPClient) WriteLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()
Loop:
	for {
		if c.status != STATUS_CONNECT {
			break Loop
		}
		select {
		case <-time.After(time.Second * time.Duration(setting.Cfg.TcpServer.CloseClientPeriod)):
			break Loop
		case msg, ok := <-c.writeCh:
			if ok {
				_, err := c.conn.Write(msg)
				if err != nil {
					log.WithField("Address", c.conn.LocalAddr()).Error(err)
					break Loop
				}
				log.WithFields(log.Fields{
					"Address": c.conn.RemoteAddr().String(),
					"Data":    fmt.Sprintf("%X", msg),
				}).Info("write data to tcp server")
			}
		}
	}
	c.Close()
}

//
func (c *TCPClient) Write(data []byte) error {
	if c.status == STATUS_CONNECT {
		c.writeCh <- data
		return nil
	} else {
		return errors.New("connection closed.")
	}
}

// 从tcp服务器读取数据
func (c *TCPClient) ReadLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	for {
		if c.status != STATUS_CONNECT {
			break
		}
		buf := make([]byte, c.readBufferSize)
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Error(err)
			break
		}

		if n > 0 {
			c.readCh <- buf[:n]
		}
	}
	c.Close()
}

// 转发TCP服务器数据至UDP客户端
func (c *TCPClient) WriteToUDPClient(conn *net.UDPConn) {
Loop:
	for {
		if c.status != STATUS_CONNECT {
			break Loop
		}
		select {
		case msg, ok := <-c.readCh:
			if ok {
				_, err := conn.WriteToUDP(msg, c.updAddr)
				if err != nil {
					log.Error("write udp client error:", err)
					break Loop
				}
				log.WithFields(log.Fields{
					"Address": c.updAddr.IP.String(),
					"Port":    c.updAddr.Port,
					"Data":    fmt.Sprintf("%X", msg),
				}).Info("write data to udp client.")
			}
		}
	}
	c.Close()
}

// 关闭相关资源
func (c *TCPClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == STATUS_CLOSE {
		return
	}
	close(c.writeCh)
	close(c.readCh)
	if c.conn != nil {
		c.conn.Close()
		log.WithField("Address", c.conn.LocalAddr()).Info("local tcp client close.")
	}
	c.status = STATUS_CLOSE
}
