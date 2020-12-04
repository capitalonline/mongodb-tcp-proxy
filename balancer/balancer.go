package balancer

import (
	"github.com/pkg/errors"
)

const (
	_mongoBalancePolicy           = 1
	_robinBalancePolicy           = 2
	_leastConnectionBalancePolicy = 3
)

type Balancer interface {
	CheckHealth()
	ChooseBackEnd() string
}

var NotSupportBalancerType = errors.New("Not support balancer type")

func CreateBalancerByType(t int, us []string, extra map[string]string) (Balancer, error) {
	switch t {
	case _mongoBalancePolicy:
		return &MongoBalancer{Upstream: us, Username: extra["username"], Password: extra["password"]}, nil
	case _robinBalancePolicy:
		return nil, NotSupportBalancerType
	case _leastConnectionBalancePolicy:
		return nil, NotSupportBalancerType
	default:
		return nil, NotSupportBalancerType
	}
}
