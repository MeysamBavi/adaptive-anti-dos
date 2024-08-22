package plan

import "sync"

type AdaptationAction func(*changes)

func AdaptLimit(newLimit float64) AdaptationAction {
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
		if c.BannedIPs == nil {
			c.BannedIPs = make(map[string]struct{})
		}
		c.BannedIPs[ip] = struct{}{}
	}
}

type changes struct {
	lock      sync.Mutex
	Limit     float64
	Replicas  int
	BannedIPs map[string]struct{}
}
