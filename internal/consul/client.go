// Package consul provides service discovery functionality using HashiCorp Consul.
// This is a shared infrastructure package used by all services (gateway, auth, posts)
// for service registration, health checks, and dynamic service discovery.
package consul

import (
	consulapi "github.com/hashicorp/consul/api"
)

// Client wraps the Consul API client
type Client struct {
	api *consulapi.Client
}

// NewClient creates a new Consul client
// func NewClient(addr string) (*Client, error) {
// 	return NewClientWithToken(addr, "")
// }

// NewClientWithToken creates a new Consul client with ACL token authentication
func NewClientWithToken(addr, token string) (*Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = addr

	// Set ACL token if provided
	if token != "" {
		config.Token = token
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Client{api: client}, nil
}

// API returns the underlying Consul API client
func (c *Client) API() *consulapi.Client {
	return c.api
}
