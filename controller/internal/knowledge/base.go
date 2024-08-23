package knowledge

type Base interface {
	CurrentLimit() float64
	CurrentReplicas() int
	SetLimit(limit int)
	SetReplicas(replicas int)
	SetBannedIPs(bannedIPs []string)
}

type impl struct {
}

func NewInMemoryBase() Base {
	return &impl{}
}

func (i *impl) CurrentLimit() float64 {
	return 7
}

func (i *impl) CurrentReplicas() int {
	return 2
}

func (i *impl) SetLimit(limit int) {
}

func (i *impl) SetReplicas(replicas int) {
}

func (i *impl) SetBannedIPs(bannedIPs []string) {
}
