package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

// ServiceConfig contains configuration for service registration
type ServiceConfig struct {
	ID      string
	Name    string
	Address string
	Port    int
	Tags    []string
	Check   *HealthCheck
}

// HealthCheck defines health check configuration
type HealthCheck struct {
	HTTP     string
	Interval string
	Timeout  string
}

// ServiceRegistrar defines the interface for service registration
type ServiceRegistrar interface {
	Register(cfg *ServiceConfig) error
	Deregister(serviceID string) error
}

// Register registers a service with Consul
func (c *Client) Register(cfg *ServiceConfig) error {
	registration := &consulapi.AgentServiceRegistration{
		ID:      cfg.ID,
		Name:    cfg.Name,
		Address: cfg.Address,
		Port:    cfg.Port,
		Tags:    cfg.Tags,
	}

	// Add health check if provided
	if cfg.Check != nil {
		registration.Check = &consulapi.AgentServiceCheck{
			HTTP:     cfg.Check.HTTP,
			Interval: cfg.Check.Interval,
			Timeout:  cfg.Check.Timeout,
		}
	}

	if err := c.api.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

// Deregister removes a service from Consul
func (c *Client) Deregister(serviceID string) error {
	if err := c.api.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	return nil
}
