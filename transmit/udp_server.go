package transmit

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gq-tang/transform/setting"
	log "github.com/sirupsen/logrus"
)

// UDP客户端集
type UDPPool struct {
	clients map[string]*TCPClient
	done    chan struct{}
	mu      sync.Mutex
	wg      sync.WaitGroup
}

// 新建客户端集
func NewUDPPool() *UDPPool {
	return &UDPPool{
		clients: make(map[string]*TCPClient),
		done:    make(chan struct{}),
	}
}

// UDP server
func (p *UDPPool) ListenUDP() {
	udpconn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: setting.Cfg.General.Port,
	})
	if err != nil {
		log.Fatal("listen udp fatal:", err)
	}
	log.Info("now listnning upd...")
	defer p.wg.Done()
	go func() {
		<-p.done
		udpconn.Close()
	}()
	for {
		readData := make([]byte, 4096)
		n, remoteAddr, err := udpconn.ReadFromUDP(readData)
		if err != nil {
			// log.Errorf("%s", err)
			break
		}
		log.WithFields(log.Fields{
			"Address": remoteAddr.IP.String(),
			"Port":    remoteAddr.Port,
			"Data":    fmt.Sprintf("%X", readData[:n]),
		}).Info("read data from udp client.")
		if client, ok := p.clients[remoteAddr.IP.String()]; ok {
			client.SetUDPAddr(remoteAddr)
			err := client.Write(readData[:n])
			if err != nil {
				log.Error(err)
			}
		} else {
			tcpAddr, err := net.ResolveTCPAddr("tcp", setting.Cfg.TcpServer.Server)
			if err != nil {
				log.Error(err)
				continue
			}
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				log.Error(err)
				continue
			}
			if setting.Cfg.TcpServer.KeepAlive {
				conn.SetKeepAlive(true)
				conn.SetKeepAlivePeriod(time.Second * time.Duration(setting.Cfg.TcpServer.KeepAlivePeriod))
			}
			client := NewTCPClient(conn, 1024, remoteAddr)
			go client.WriteLoop()
			go client.ReadLoop()
			go client.WriteToUDPClient(udpconn)
			p.clients[remoteAddr.IP.String()] = client
			err = client.Write(readData[:n])
			if err != nil {
				log.Error(err)
			}
		}
	}
}

// clear invalid client
func (p *UDPPool) clear() {
	defer p.wg.Done()
Loop:
	for {
		select {
		case <-p.done:
			break Loop
		case <-time.After(time.Second):
			for key, val := range p.clients {
				if val.status == STATUS_CLOSE {
					delete(p.clients, key)
				}
			}
		}
	}
}

func (p *UDPPool) Server() {
	log.WithFields(log.Fields{
		"Port":      setting.Cfg.General.Port,
		"TCPServer": setting.Cfg.TcpServer.Server,
	}).Info("Start server")
	p.wg.Add(2)
	go p.clear()
	go p.ListenUDP()
}

func (p *UDPPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, client := range p.clients {
		client.Close()
		delete(p.clients, key)
	}
	close(p.done)
	// 等待相关资源关闭
	p.wg.Wait()
}
