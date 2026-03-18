package loadbalancer

type BalancerConfig struct {
	Path     string
	Balancer string
	Servers  []string
}
