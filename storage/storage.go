package storage

import (
	"errors"
	"mongodb-proxy/conf"
	"mongodb-proxy/proxy"
)

const (
	_consulStorageType = "consul"
	_mysqlStorageType  = "mysql"
)

var NotSupportStorageType = errors.New("not support storage type")

// TODO 存储模块 进出直接是proxy 移除ConsulData
type Storage interface {
	Add(metaData *proxy.Proxy) error
	Delete(id string) error
	Scan() (services []*proxy.Proxy, err error)
	Watch(id string) (data *proxy.Proxy, err error)
	Observe(handle func(index uint64, result interface{}))
}

func NewStorage(c *conf.Storage) (Storage, error) {
	switch c.Type {
	case _consulStorageType:
		return NewConsulStroage(c.DSN), nil
	case _mysqlStorageType:
		return nil, NotSupportStorageType
	default:
		return nil, NotSupportStorageType
	}
}
