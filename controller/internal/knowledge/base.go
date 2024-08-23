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
	CurrentBannedIPs() ([]string, time.Time)
	SetBannedIPs(ips []string)
}

type impl struct {
	limit                atomic.Int32
	replicas             atomic.Int32
	pendingReplicaChange atomic.Bool
	pendingLimitChange   atomic.Bool
	bannedIPsLock        sync.RWMutex
	bannedIPs            []string
	banTime              atomic.Int64
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

func (i *impl) CurrentBannedIPs() ([]string, time.Time) {
	i.bannedIPsLock.RLock()
	defer i.bannedIPsLock.RUnlock()
	return i.bannedIPs, time.Unix(i.banTime.Load(), 0)
}

func (i *impl) SetBannedIPs(ips []string) {
	i.bannedIPsLock.Lock()
	defer i.bannedIPsLock.Unlock()
	i.bannedIPs = ips
	i.banTime.Store(time.Now().Unix())
}
