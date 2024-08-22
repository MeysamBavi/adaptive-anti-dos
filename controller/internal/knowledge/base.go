package knowledge

type Base interface {
	CurrentLimit() float64
	CurrentReplicas() int
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
