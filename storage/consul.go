package storage

import (
	"encoding/json"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/sirupsen/logrus"
	"mongodb-proxy/proxy"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

type ConsulStorage struct {
	Addr   string
	client *consul.Client
}

func NewConsulStroage(addr string) *ConsulStorage {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	wrapper := &http.Client{
		Transport: transport,
		Timeout:   35 * time.Second,
	}
	clientConf := &consul.Config{
		Address:    addr,
		Scheme:     "http",
		HttpClient: wrapper,
	}
	client, err := consul.NewClient(clientConf)
	if err != nil {
		panic(err)
	}

	return &ConsulStorage{
		Addr:   addr,
		client: client,
	}

}

/*服务注册
  metaData: 服务注册所需要的参数
  server: consul地址+端口  101.251.219.226:8500
*/
func (cs *ConsulStorage) Add(metaData *proxy.Proxy) error {
	registration := new(consul.AgentServiceRegistration)
	registration.ID = metaData.Name // 服务节点的名称
	registration.Name = "proxyNode" // 服务名称
	port, err := strconv.Atoi(metaData.Extra["port"])
	if err != nil {
		logrus.Error("Upstream port error : ", err)
		return err
	}
	registration.Port = port                         // 服务端口
	registration.Address = metaData.Extra["address"] // 服务 IP
	registration.Tags = []string{"proxyNode"}
	upstream, err := json.Marshal(metaData.Upstream)
	if err != nil {
		logrus.Error("Upstream error : ", err)
		return err
	}
	registration.Meta = map[string]string{
		"customer_id":     metaData.Extra["customer_id"],
		"type":            "proxy",
		"user_id":         metaData.Extra["user_id"],
		"listen":          metaData.Listen,
		"upstream":        string(upstream),
		"enabled":         strconv.FormatBool(metaData.Enabled),
		"username":        metaData.Extra["username"],
		"password":        metaData.Extra["password"],
		"port":            metaData.Extra["port"],
		"role":            metaData.Extra["role"],
		"replicaset_name": metaData.Extra["replicaset_name"],
	}
	RegErr := cs.client.Agent().ServiceRegister(registration)
	if RegErr != nil {
		logrus.Error("add register server error : ", RegErr)
		return RegErr
	}
	return nil
}

/*
   服务删除
   id:consul服务id
*/
func (cs *ConsulStorage) Delete(id string) error {
	err := cs.client.Agent().ServiceDeregister(id)
	if err != nil {
		logrus.Error("delete register server error : ", err)
		return err
	}
	return nil
}

/*
   获取所有服务
*/
func (cs *ConsulStorage) Scan() (services []*proxy.Proxy, err error) {
	agentService, err := cs.client.Agent().ServicesWithFilter("proxyNode in Tags")
	if err != nil {
		logrus.Error("scan register server error : ", err)
		return nil, err
	}
	for name, agent := range agentService {
		var upstream []string
		json.Unmarshal([]byte(agent.Meta["upstream"]), &upstream)
		enabled, _ := strconv.ParseBool(agent.Meta["enabled"])

		data := &proxy.Proxy{
			Name:     name,
			Listen:   agent.Meta["listen"],
			Upstream: upstream,
			Enabled:  enabled,
			Extra: map[string]string{
				"username":        agent.Meta["username"],
				"password":        agent.Meta["password"],
				"customer_id":     agent.Meta["customer_id"],
				"user_id":         agent.Meta["user_id"],
				"port":            agent.Meta["port"],
				"role":            agent.Meta["role"],
				"replicaset_name": agent.Meta["replicaset_name"],
			},
		}
		services = append(services, data)
	}
	return services, nil
}

/*
   获取当前服务
   id:consul服务id
*/
func (cs *ConsulStorage) Watch(id string) (data *proxy.Proxy, err error) {
	agent, _, err := cs.client.Agent().Service(id, nil)
	if err != nil {
		logrus.Error("register server error : ", err)
		return nil, err
	}
	var upstream []string
	json.Unmarshal([]byte(agent.Meta["upstream"]), &upstream)
	enabled, _ := strconv.ParseBool(agent.Meta["enabled"])
	data = &proxy.Proxy{
		//Mutex:    sync.Mutex{},
		Name:     agent.ID,
		Listen:   agent.Meta["listen"],
		Upstream: upstream,
		Enabled:  enabled,
		Extra: map[string]string{
			"username":        agent.Meta["username"],
			"password":        agent.Meta["password"],
			"customer_id":     agent.Meta["customer_id"],
			"user_id":         agent.Meta["user_id"],
			"port":            agent.Meta["port"],
			"role":            agent.Meta["role"],
			"replicaset_name": agent.Meta["replicaset_name"],
		},
	}
	return data, nil
}

func (cs *ConsulStorage) Observe(handle func(index uint64, result interface{})) {
	params := make(map[string]interface{})
	params["type"] = "service"
	params["service"] = "proxyNode"
	params["passingonly"] = false
	params["tag"] = "proxyNode"
	plan, err := watch.Parse(params)
	if err != nil {
		panic(err)
	}
	plan.Handler = handle
	if err = plan.Run(cs.Addr); err != nil {
		panic(err)
	}
	logrus.WithFields(logrus.Fields{
		"addr": cs.Addr,
	}).Info("Begin observe")
}

//添加、删除、获取所有、run
type ConsulData struct {
	CustomerId   string            `json:"customer_id"`
	UserId       string            `json:"user_id"`
	Listen       string            `json:"listen"`
	Enabled      bool              `json:"enabled"`
	Extra        map[string]string `json:"extra"`
	ClusterID    string            `json:"cluster_id"`
	Upstream     []string          `json:"upstream"`
	ReplSetName  string            `json:"replicaset_name"`
	Role         string            `json:"role"`
	Port         int               `json:"port"`
	ProxyAddress string            `json:"proxy_address"`
	ProxyName    string            `json:"proxy_name"`
}
