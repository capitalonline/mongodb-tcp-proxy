package mongodb_proxy

import (
	"strconv"
	"testing"
	"time"
)

func TestLink(t *testing.T) {

	m := make(map[string]string)
	for i := 0; i < 5000; i++ {
		m[strconv.Itoa(i)] = strconv.Itoa(i)
	}

	for i := 0; i < 5000; i++ {
		go func(a int) {
			delete(m, strconv.Itoa(a))
		}(i)
	}
	time.Sleep(30 * time.Second)

}
