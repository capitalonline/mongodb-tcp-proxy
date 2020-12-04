package mongodb_proxy

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"log"
	"mongodb-proxy/balancer"
	"mongodb-proxy/conf"
	proxy2 "mongodb-proxy/proxy"
	"net"
	"net/http"
)

var (
	ErrBadRequestBody     = newError("bad request body", http.StatusBadRequest)
	ErrMissingField       = newError("missing required field", http.StatusBadRequest)
	ErrProxyNotFound      = newError("proxy not found", http.StatusNotFound)
	ErrProxyAlreadyExists = newError("proxy already exists", http.StatusConflict)
	ErrInvalidStream      = newError("stream was invalid, can be either upstream or downstream", http.StatusBadRequest)
)

type RestEndpoint struct {
	Host          string
	Port          string
	PolicyManager *ProxyManager
}

func NewRestEndpoint(p *ProxyManager, conf *conf.App) *RestEndpoint {
	return &RestEndpoint{PolicyManager: p, Host: conf.Host, Port: conf.Port}
}

func (ep *RestEndpoint) ProxyCreate(w http.ResponseWriter, r *http.Request) {
	input := proxy2.Proxy{Enabled: true}
	err := json.NewDecoder(r.Body).Decode(&input)
	if apiError(w, joinError(err, ErrBadRequestBody)) {
		return
	}

	if len(input.Name) < 1 {
		apiError(w, joinError(fmt.Errorf("name"), ErrMissingField))
		return
	}
	if len(input.Upstream) < 1 {
		apiError(w, joinError(fmt.Errorf("upstream"), ErrMissingField))
		return
	}

	proxy := proxy2.NewProxy()
	//name clusterid+"_proxy"
	proxy.Name = input.Name
	proxy.Listen = input.Listen
	proxy.Upstream = input.Upstream
	proxy.Extra = input.Extra
	proxy.Balancer, _ = balancer.CreateBalancerByType(1, proxy.Upstream, proxy.Extra)

	err = ep.PolicyManager.Add(proxy)
	if apiError(w, err) {
		return
	}
	data, err := json.Marshal(proxy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(data)
	if err != nil {
		logrus.Warn("ProxyCreate: Failed to write response to client", err)
	}
}

func (ep *RestEndpoint) ProxyDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := ep.PolicyManager.Remove(vars["proxy"])
	if apiError(w, err) {
		return
	}
	str := map[string]string{"ok": "deleted"}
	data, err := json.Marshal(str)
	if apiError(w, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	if err != nil {
		logrus.Warn("ProxyDelete: Failed to write headers to client", err)
	}
}

func (ep *RestEndpoint) ProxyShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proxy, err := ep.PolicyManager.Get(vars["proxy"])
	if apiError(w, err) {
		return
	}
	//data, err := json.Marshal(proxy)
	//if apiError(w, err) {
	//	return
	//}
	consulData, err := ep.PolicyManager.stg.Watch(proxy.Name)
	if err != nil {
		logrus.Warn("ProxyShow on consul: Failed to write response to client", err)
	}
	allData := make(map[string]interface{})
	allData["map"] = proxy
	allData["consul"] = consulData
	data, err := json.Marshal(allData)
	if apiError(w, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		logrus.Warn("ProxyShow: Failed to write response to client", err)
	}
}

func (ep *RestEndpoint) ProxyUpdate(w http.ResponseWriter, r *http.Request) {

}

func (ep *RestEndpoint) ConnectionShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proxy, err := ep.PolicyManager.Get(vars["proxy"])
	if apiError(w, err) {
		return
	}

	render := map[string]map[string]interface{}{
		"upstream_connection": {
			"number": len(proxy.UpstreamConnections().List()),
			"list":   proxy.UpstreamConnections().List(),
		},
		"downstream_connection": {
			"number": len(proxy.DownstreamConnection().List()),
			"list":   proxy.DownstreamConnection().List(),
		},
		"link": {
			"number": len(proxy.Links().Links()),
			"list":   proxy.Links().Links(),
		},
	}

	data, err := json.Marshal(render)
	if apiError(w, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		logrus.Warn("ProxyShow: Failed to write response to client", err)
	}
}

func (ep *RestEndpoint) Ping(w http.ResponseWriter, r *http.Request) {
	//s := r.String(200, "pong")
	str := map[string]string{"ok": "pong"}
	data, err := json.Marshal(str)
	if apiError(w, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		logrus.Warn("ProxyShow: Failed to write response to client", err)
	}
}

func (ep *RestEndpoint) Run() {
	r := mux.NewRouter()
	r.HandleFunc("/proxies", ep.ProxyCreate).Methods("POST")
	r.HandleFunc("/proxies/{proxy}", ep.ProxyShow).Methods("GET")
	r.HandleFunc("/proxies/{proxy}", ep.ProxyUpdate).Methods("POST")
	r.HandleFunc("/proxies/{proxy}", ep.ProxyDelete).Methods("DELETE")
	r.HandleFunc("/proxies/{proxy}/connections", ep.ConnectionShow).Methods("GET")
	r.HandleFunc("/health", ep.Ping).Methods("GET")

	logrus.WithFields(logrus.Fields{
		"host": ep.Host,
		"port": ep.Port,
	}).Info("API HTTP server starting")

	err := http.ListenAndServe(net.JoinHostPort(ep.Host, ep.Port), r)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// inner

func apiError(resp http.ResponseWriter, err error) bool {
	obj, ok := err.(*ApiError)
	if !ok && err != nil {
		logrus.Warn("Error did not include status code: ", err)
		obj = &ApiError{err.Error(), http.StatusInternalServerError}
	}

	if obj == nil {
		return false
	}

	data, err2 := json.Marshal(obj)
	if err2 != nil {
		logrus.Warn("Error json encoding error (╯°□°）╯︵ ┻━┻ ", err2)
	}
	resp.Header().Set("Content-Type", "application/json")
	http.Error(resp, string(data), obj.StatusCode)

	return true
}

type ApiError struct {
	Message    string `json:"error"`
	StatusCode int    `json:"status"`
}

func (e *ApiError) Error() string {
	return e.Message
}

func newError(msg string, status int) *ApiError {
	return &ApiError{msg, status}
}

func joinError(err error, wrapper *ApiError) *ApiError {
	if err != nil {
		return &ApiError{wrapper.Message + ": " + err.Error(), wrapper.StatusCode}
	}
	return nil
}
