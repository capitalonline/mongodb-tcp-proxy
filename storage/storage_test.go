package storage

import (
	"fmt"
	"mongodb-proxy/proxy"

	"testing"
)

func TestCreateConsul(t *testing.T) {
	//data1 := ConsulData{
	//	CustomerId: "11",
	//	UserId:     "11",
	//	Listen:     "0.0.0.0:27080",
	//	Enabled:    true,
	//	Extra: map[string]string{
	//		"username": "test",
	//		"password": "123qwe",
	//	},
	//	ClusterID:    "123",
	//	Upstream:     []string{"101.251.220.22:27017", "101.251.220.23:27017", "101.251.220.20:27017"},
	//	ReplSetName:  "test",
	//	Role:         "ad",
	//	Port:         222,
	//	ProxyAddress: "202.202.0.29",
	//}

	//data := ConnsulData{
	//	CustomerId:   "111",
	//	UserId:       "11",
	//	VmId:         "11",
	//	ClusterID:    "33",
	//	UserIP:       "1",
	//	ReplSetName:  "11",
	//	Role:         "11",
	//	Port:         1230,
	//	ProxyAddress: "202.202.0.29",
	//}
	data := &proxy.Proxy{
		Name:     "sf9c4abc-ec18-4594-afa7-b425658d6c5_proxy",
		Listen:   "0.0.0.0:8994",
		Upstream: nil,
		Enabled:  false,
		Extra: map[string]string{
			"username":        "test123",
			"password":        "123321",
			"customer_id":     "123312",
			"user_id":         "123",
			"port":            "123",
			"role":            "123",
			"replicaset_name": "123",
		},
	}
	stroage := NewConsulStroage("101.251.219.226:8500")
	stroage.Add(data)
	//CreateConsul(data, "101.251.219.226:8500")
}

func TestDeleteConsul(t *testing.T) {
	stroage := NewConsulStroage("101.251.219.226:8500")
	stroage.Delete("asdasdasdasdasdas")

}

func TestScanConsul(t *testing.T) {
	stroage := NewConsulStroage("101.251.219.226:8500")
	services, err := stroage.Scan()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(services)
}

func TestConsulStorage_Watch(t *testing.T) {
	stroage := NewConsulStroage("101.251.219.226:8500")
	data, err := stroage.Watch("2f9c4abc-ec18-4594-afa7-b425658d6c5_proxy")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data)
}
