package balancer

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type MongoBalancer struct {
	Username string //加密
	Password string // 加密
	Upstream []string
	current  string
}

func (lb *MongoBalancer) CheckHealth() {

}

func (lb *MongoBalancer) ChooseBackEnd() string {
	//if lb.current != "" {
	//	return lb.current
	//}
	for _, upstream := range lb.Upstream {
		if isMasterSync(upstream, lb.Username, lb.Password) {
			lb.current = upstream
			return upstream
		}
	}
	return ""
}

func isMasterSync(addr string, user string, pwd string) bool {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	opt := options.Client()
	opt.Direct = toBoolPtr(true)
	opt.ApplyURI("mongodb://" + user + ":" + pwd + "@" + addr + "/admin")
	//client, err := mongo.NewClient(opt)
	client, err := mongo.Connect(ctx, opt)
	if err != nil {
		logrus.Println(err)
		return false
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Nearest())
	if err != nil {
		logrus.Println(err)
		return false
	}
	result := client.Database("admin").RunCommand(ctx, bson.M{"isMaster": 1})
	var rst bson.M
	result.Decode(&rst)

	return rst["ismaster"].(bool)
}

func toBoolPtr(b bool) *bool {
	return &b
}
