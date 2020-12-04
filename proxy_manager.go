package mongodb_proxy

import (
	consul "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"math/rand"
	"mongodb-proxy/balancer"
	"mongodb-proxy/conf"
	"mongodb-proxy/proxy"
	sg "mongodb-proxy/storage"
	"sync"
	"time"
)

// ProxyManager needs to check the service health regularly,
// and election by raft
type ProxyManager struct {
	sync.RWMutex
	proxies map[string]*proxy.Proxy
	stg     sg.Storage
}

func NewProxyManager(c *conf.Config) *ProxyManager {
	storage, err := sg.NewStorage(c.Storage)
	if err != nil {
		panic(err)
	}
	return &ProxyManager{
		proxies: make(map[string]*proxy.Proxy),
		stg:     storage,
	}
}

func (pm *ProxyManager) Start() {
	services, err := pm.stg.Scan()
	if err != nil {
		logrus.Warn("Proxy scan on consul: Failed ", err)
	}
	for _, service := range services {
		proxy := proxy.NewProxy()
		proxy.Name = service.Name
		proxy.Listen = service.Listen
		proxy.Upstream = service.Upstream
		proxy.Extra = service.Extra
		//proxy.Enabled = service.Enabled
		proxy.Balancer, _ = balancer.CreateBalancerByType(1, proxy.Upstream, proxy.Extra)
		addErr := pm.Add(proxy)
		if addErr != nil {
			logrus.Warn("add Proxy on consul: Failed", err)
		}

	}
	//ohandle := func(index uint64, result interface{}) {
	//	if entries, ok := result.([]*consul.ServiceEntry); ok {
	//		// 获取差量
	//		currentProxy := set.New(set.ThreadSafe)
	//		changeProxy := set.New(set.ThreadSafe)
	//		for _, current := range keys(pm.proxies) {
	//			currentProxy.Add(current)
	//		}
	//		for _, change := range consulKeys(entries) {
	//			changeProxy.Add(change)
	//		}
	//
	//		//差集
	//		deleteProxy := set.Difference(currentProxy, changeProxy)
	//		addProxy := set.Difference(changeProxy, currentProxy)
	//
	//		for _, dp := range deleteProxy.List() {
	//			rerr := pm.Remove(dp.(string))
	//			if rerr != nil {
	//				logrus.WithFields(logrus.Fields{
	//					"proxy_name": dp.(string),
	//					"err":        rerr,
	//				}).Error("remove proxy")
	//			}
	//		}
	//		for _, entry := range entries {
	//
	//			if addProxy.Has(entry.Service.ID) {
	//				var upstream []string
	//				json.Unmarshal([]byte(entry.Service.Meta["upstream"]), &upstream)
	//
	//				extra := map[string]string{
	//					"username":        entry.Service.Meta["username"],
	//					"password":        entry.Service.Meta["password"],
	//					"customer_id":     entry.Service.Meta["customer_id"],
	//					"user_id":         entry.Service.Meta["user_id"],
	//					"port":            entry.Service.Meta["port"],
	//					"role":            entry.Service.Meta["role"],
	//					"replicaset_name": entry.Service.Meta["replicaset_name"],
	//				}
	//
	//				newProxy := proxy.NewProxy()
	//				newProxy.Name = entry.Service.ID
	//				newProxy.Listen = entry.Service.Meta["listen"]
	//				newProxy.Upstream = upstream
	//				newProxy.Extra = extra
	//				newProxy.Balancer, _ = balancer.CreateBalancerByType(1, newProxy.Upstream, newProxy.Extra)
	//				addErr := pm.Add(newProxy)
	//				if addErr != nil {
	//					logrus.Warn("add Proxy on consul: Failed", err)
	//				}
	//
	//			}
	//
	//		}
	//	}
	//}
	//go pm.stg.Observe(ohandle)

}

func (pm *ProxyManager) Add(proxy *proxy.Proxy) error {
	pm.Lock()

	defer pm.Unlock()
	if _, exists := pm.proxies[proxy.Name]; exists {
		return ErrProxyAlreadyExists
	}
	err := proxy.Start()
	if err != nil {
		return err
	}
	//add consul
	addErr := Retry(3, 500*time.Millisecond, func() error {
		err = pm.stg.Add(proxy)
		if err != nil {
			return err
		}
		return nil
	})
	if addErr != nil {
		logrus.Warn("ProxyCreate on consul: Failed to write response to client", err)
	}
	pm.proxies[proxy.Name] = proxy
	return nil
}

func (pm *ProxyManager) AddOrReplace() {
	pm.Lock()
	defer pm.Unlock()
}

func (pm *ProxyManager) Proxies() {
	pm.Lock()
	defer pm.Unlock()

}

func (pm *ProxyManager) Clear() {
	pm.Lock()
	defer pm.Unlock()

}

func (pm *ProxyManager) Remove(name string) error {
	pm.Lock()
	defer pm.Unlock()
	targetProxy, err := pm.getByName(name)
	if err != nil {
		return err
	}

	targetProxy.Stop()
	delete(pm.proxies, targetProxy.Name)
	delErr := Retry(3, 500*time.Millisecond, func() error {
		err = pm.stg.Delete(name)
		if err != nil {
			return err
		}
		return nil
	})
	if delErr != nil {
		logrus.Warn("ProxyDelete on consul: Failed to write headers to client", err)
	}
	return nil
}

func (pm *ProxyManager) Get(name string) (*proxy.Proxy, error) {
	pm.Lock()
	defer pm.Unlock()
	return pm.getByName(name)
}

func (pm *ProxyManager) getByName(name string) (*proxy.Proxy, error) {
	// for inner, not lock
	proxy, exists := pm.proxies[name]
	if !exists {
		return nil, ErrProxyNotFound
	}
	return proxy, nil
}

func keys(m map[string]*proxy.Proxy) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func consulKeys(ents []*consul.ServiceEntry) []string {
	keys := make([]string, 0, len(ents))
	for _, ent := range ents {
		keys = append(keys, ent.Service.ID)
	}
	return keys
}
func Retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			r := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + r/2

			time.Sleep(sleep)
			return Retry(attempts, 2*sleep, f)
		}
		return err
	}
	return nil
}
