package loadbalancer

type Backend string

type Balancer interface {
	NextBackend() (Backend, error)
}

func ResolveBalancer(cfg BalancerConfig) Balancer {
	switch cfg.Balancer {
	case "roundrobin":
		return NewRoundRobin(cfg.Servers)
	default:
		return NewRoundRobin(cfg.Servers)
	}
}
