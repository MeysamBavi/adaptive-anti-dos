package plan

import (
	"net"
	"sync"
)

type AdaptationAction func(*changes)

func AdaptLimit(newLimit int) AdaptationAction {
	return func(c *changes) {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.Limit = newLimit
	}
}

func AdaptReplicas(newReplicas int) AdaptationAction {
	return func(c *changes) {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.Replicas = newReplicas
	}
}

func BanIP(ip string) AdaptationAction {
	return func(c *changes) {
		c.lock.Lock()
		defer c.lock.Unlock()
		if v := net.ParseIP(ip); v != nil {
			c.BanOrUnban[v.String()] = true
		}
	}
}

func UnbanIP(ip string) AdaptationAction {
	return func(c *changes) {
		c.lock.Lock()
		defer c.lock.Unlock()
		if v := net.ParseIP(ip); v != nil {
			c.BanOrUnban[v.String()] = false
		}
	}
}

type changes struct {
	lock       sync.Mutex
	Limit      int
	Replicas   int
	BanOrUnban map[string]bool
}
