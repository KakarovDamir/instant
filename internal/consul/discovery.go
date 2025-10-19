package consul

import (
	"fmt"
	"math/rand"
)

// ServiceInstance represents a discovered service instance
type ServiceInstance struct {
	ID      string
	Name    string
	Address string
	Port    int
	Tags    []string
}

// ServiceDiscovery defines the interface for service discovery
type ServiceDiscovery interface {
	Discover(serviceName string) ([]*ServiceInstance, error)
	DiscoverOne(serviceName string) (*ServiceInstance, error)
}

// Discover retrieves all healthy instances of a service
func (c *Client) Discover(serviceName string) ([]*ServiceInstance, error) {
	// Query for healthy services only
	services, _, err := c.api.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	// Convert to ServiceInstance slice
	instances := make([]*ServiceInstance, 0, len(services))
	for _, entry := range services {
		instance := &ServiceInstance{
			ID:      entry.Service.ID,
			Name:    entry.Service.Service,
			Address: entry.Service.Address,
			Port:    entry.Service.Port,
			Tags:    entry.Service.Tags,
		}

		// Use node address if service address is empty
		if instance.Address == "" {
			instance.Address = entry.Node.Address
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

// DiscoverOne retrieves a single healthy instance using random load balancing
func (c *Client) DiscoverOne(serviceName string) (*ServiceInstance, error) {
	instances, err := c.Discover(serviceName)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances available for service: %s", serviceName)
	}

	// Simple random load balancing
	idx := rand.Intn(len(instances))
	return instances[idx], nil
}

// DiscoverCatalog returns all registered services (not just healthy ones)
func (c *Client) DiscoverCatalog(serviceName string, tag string) ([]*ServiceInstance, error) {
	services, _, err := c.api.Catalog().Service(serviceName, tag, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service from catalog: %w", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no instances found in catalog for service: %s", serviceName)
	}

	instances := make([]*ServiceInstance, 0, len(services))
	for _, svc := range services {
		instance := &ServiceInstance{
			ID:      svc.ServiceID,
			Name:    svc.ServiceName,
			Address: svc.ServiceAddress,
			Port:    svc.ServicePort,
			Tags:    svc.ServiceTags,
		}

		// Use node address if service address is empty
		if instance.Address == "" {
			instance.Address = svc.Address
		}

		instances = append(instances, instance)
	}

	return instances, nil
}
