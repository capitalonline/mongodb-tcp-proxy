package proxy

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/tomb.v2"
	"io"
	"mongodb-proxy/balancer"
	"mongodb-proxy/stream"
	"mongodb-proxy/utils"
	"net"
	"strconv"
	"sync"
	"time"
)

type Proxy struct {
	sync.Mutex
	//存放ruleid, Name为了更加通用
	Name     string   `json:"name"`
	Listen   string   `json:"listen"`
	Upstream []string `json:"upstream"`
	Enabled  bool     `json:"enabled"`
	//存放用户名和密码
	Extra   map[string]string `json:"extra"`
	started chan error

	tomb                 tomb.Tomb
	upstreamConnections  ConnectionList
	downstreamConnection ConnectionList
	links                *LinkList

	Balancer balancer.Balancer
}

func (p *Proxy) Links() *LinkList {
	return p.links
}

func (p *Proxy) DownstreamConnection() ConnectionList {
	return p.downstreamConnection
}

func (p *Proxy) UpstreamConnections() ConnectionList {
	return p.upstreamConnections
}

type Link struct {
	Name       string
	Proxy      *Proxy
	LinkList   *LinkList
	Upstream   net.Conn
	Downstream net.Conn
	direction  stream.Direction
}

func (lk *Link) Run() {
	logrus.WithFields(logrus.Fields{
		"name":       lk.Name,
		"proxy":      lk.Proxy.Name,
		"upstream":   lk.Upstream.RemoteAddr(),
		"downstream": lk.Downstream.RemoteAddr(),
	}).Info("Link downstream to upstream")
	go func() {
		bytes, err := io.Copy(lk.Upstream, lk.Downstream)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"name":  lk.Name,
				"bytes": bytes,
				"err":   err,
			}).Warn("Source terminated")
		}
		lk.Upstream.Close()
		lk.Proxy.RemoveUpstreamConnection(lk.Name + "-upstream")

	}()
	go func() {
		bytes, err := io.Copy(lk.Downstream, lk.Upstream)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"name":  lk.Name,
				"proxy": lk.Proxy.Name,
				"bytes": bytes,
				"err":   err,
			}).Warn("Destination terminated")
		}
		lk.Downstream.Close()
		lk.Proxy.RemoveDownstreamConnection(lk.Name + "-downstream")
		lk.LinkList.RemoveLink(lk.Name)
		//delete(lk.Proxy.links.links, lk.Name)

	}()
}

type LinkList struct {
	sync.Mutex
	proxy *Proxy
	links map[string]*Link
}

func (L *LinkList) Links() map[string]*Link {
	return L.links
}

func CreateLinkList(p *Proxy) *LinkList {
	return &LinkList{
		proxy: p,
		links: make(map[string]*Link),
	}
}

func (L *LinkList) StartLink(name string, upstream, downstream net.Conn) {
	L.Lock()
	defer L.Unlock()
	lk := &Link{
		Name:       name,
		Proxy:      L.proxy,
		LinkList:   L,
		Upstream:   upstream,
		Downstream: downstream,
		direction:  0,
	}
	lk.Run()
	L.links[name] = lk
}

func (L *LinkList) RemoveLink(name string) {
	L.Lock()
	defer L.Unlock()
	delete(L.links, name)
}

type ConnectionList struct {
	list map[string]net.Conn
	lock sync.Mutex
}

func (c ConnectionList) List() map[string]net.Conn {
	return c.list
}

func (c *ConnectionList) Lock() {
	c.lock.Lock()
}

func (c *ConnectionList) Unlock() {
	c.lock.Unlock()
}

var ErrProxyAlreadyStarted = errors.New("Proxy already started")

func NewProxy() *Proxy {
	//byType, err := balancer.CreateBalancerByType()
	proxy := &Proxy{
		started:              make(chan error),
		upstreamConnections:  ConnectionList{list: make(map[string]net.Conn)},
		downstreamConnection: ConnectionList{list: make(map[string]net.Conn)},
	}
	proxy.links = CreateLinkList(proxy)
	return proxy
}

func (p *Proxy) Start() error {
	p.Lock()
	defer p.Unlock()

	return start(p)
}

func (p *Proxy) Stop() {
	p.Lock()
	defer p.Unlock()
	stop(p)
}

func (p *Proxy) RemoveUpstreamConnection(name string) {
	p.upstreamConnections.Lock()
	defer p.upstreamConnections.Unlock()
	delete(p.upstreamConnections.list, name)
}
func (p *Proxy) RemoveDownstreamConnection(name string) {
	p.downstreamConnection.Lock()
	defer p.downstreamConnection.Unlock()
	delete(p.downstreamConnection.list, name)
}

func (p *Proxy) server() {
	boss, err := net.Listen("tcp", p.Listen)
	if err != nil {
		p.started <- err
		return
	}

	p.Listen = boss.Addr().String()
	p.started <- nil

	logrus.WithFields(logrus.Fields{
		"name":     p.Name,
		"proxy":    p.Listen,
		"upstream": p.Upstream,
	}).Info("Start proxy")

	//acceptTomb := tomb.Tomb{}
	//defer acceptTomb.Dead()

	go func() {
		<-p.tomb.Dying()

		// Notify ln.Accept() that the shutdown was safe
		//acceptTomb.Killf("Shutting down from stop()")
		// Unblock ln.Accept()
		err := boss.Close()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"proxy":  p.Name,
				"listen": p.Listen,
				"err":    err,
			}).Warn("Attempted to close an already closed proxy server")
		}

		// Wait for the accept loop to finish processing
		p.tomb.Dead()
	}()

	for {
		connection, err := boss.Accept()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"name":     p.Name,
				"proxy":    p.Listen,
				"upstream": p.Upstream,
				"err":      err,
			}).Error("Accepted client")
			return
		}
		logrus.WithFields(logrus.Fields{
			"name":     p.Name,
			"client":   connection.RemoteAddr(),
			"proxy":    p.Listen,
			"upstream": p.Upstream,
		}).Info("Accepted client")

		ip := p.Balancer.ChooseBackEnd()
		if ip == "" {
			connection.Close()
			continue
		}

		upstream, err := net.Dial("tcp", ip)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"name":     p.Name,
				"client":   connection.RemoteAddr(),
				"proxy":    p.Listen,
				"upstream": p.Upstream,
			}).Error("Unable to open connection to upstream")
			connection.Close()
			continue
		}

		name := connection.RemoteAddr().String() + strconv.FormatInt(time.Now().Unix(), 10) + utils.RandStringRunes(6)
		// TODO 使用lock-free优化
		p.upstreamConnections.Lock()
		p.upstreamConnections.list[name+"-upstream"] = upstream
		p.upstreamConnections.Unlock()
		p.downstreamConnection.Lock()
		p.downstreamConnection.list[name+"-downstream"] = connection
		p.downstreamConnection.Unlock()
		p.links.StartLink(name, upstream, connection)
	}
}

func start(p *Proxy) error {
	if p.Enabled {
		return ErrProxyAlreadyStarted
	}
	p.tomb = tomb.Tomb{} // Reset tomb, from previous starts/stops
	go p.server()
	err := <-p.started
	p.Enabled = err == nil
	return err
}

func stop(p *Proxy) {
	if !p.Enabled {
		return
	}
	p.Enabled = false

	p.tomb.Killf("Shutting down from stop()")
	//p.tomb.Wait()

	p.downstreamConnection.Lock()
	defer p.downstreamConnection.Unlock()
	for _, conn := range p.downstreamConnection.list {
		conn.Close()
	}

	p.upstreamConnections.Lock()
	defer p.upstreamConnections.Unlock()
	for _, conn := range p.upstreamConnections.list {
		conn.Close()
	}

}
