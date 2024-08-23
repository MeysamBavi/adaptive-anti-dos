package knowledge

import (
	"sync"
	"sync/atomic"
	"time"
)

type Base interface {
	CurrentLimit() int
	CurrentReplicas() int
	SetLimit(limit int)
	SetReplicas(replicas int)
	SetPendingLimitChange(bool)
	SetPendingReplicaChange(bool)
	HasPendingLimitChange() bool
	HasPendingReplicaChange() bool
	RangeBannedIPs(func(string, time.Time))
	BanIP(ip string)
	UnbanIP(ip string)
}

type impl struct {
	limit                atomic.Int32
	replicas             atomic.Int32
	pendingReplicaChange atomic.Bool
	pendingLimitChange   atomic.Bool
	bannedIPs            sync.Map
}

func NewInMemoryBase() Base {
	return &impl{}
}

func (i *impl) CurrentLimit() int {
	return int(i.limit.Load())
}

func (i *impl) CurrentReplicas() int {
	return int(i.replicas.Load())
}

func (i *impl) SetLimit(limit int) {
	i.limit.Store(int32(limit))
	i.SetPendingLimitChange(false)
}

func (i *impl) SetReplicas(replicas int) {
	i.replicas.Store(int32(replicas))
	i.SetPendingReplicaChange(false)
}

func (i *impl) SetPendingReplicaChange(b bool) {
	i.pendingReplicaChange.Store(b)
}

func (i *impl) SetPendingLimitChange(b bool) {
	i.pendingLimitChange.Store(b)
}

func (i *impl) HasPendingReplicaChange() bool {
	return i.pendingReplicaChange.Load()
}

func (i *impl) HasPendingLimitChange() bool {
	return i.pendingLimitChange.Load()
}

func (i *impl) RangeBannedIPs(f func(string, time.Time)) {
	i.bannedIPs.Range(func(k, v any) bool {
		f(k.(string), v.(time.Time))
		return true
	})
}

func (i *impl) BanIP(ip string) {
	i.bannedIPs.LoadOrStore(ip, time.Now())
}

func (i *impl) UnbanIP(ip string) {
	i.bannedIPs.Delete(ip)
}
