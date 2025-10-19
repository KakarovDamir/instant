package gateway

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"instant/internal/consul"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles reverse proxy requests to backend services
type ProxyHandler struct {
	discovery consul.ServiceDiscovery
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(discovery consul.ServiceDiscovery) *ProxyHandler {
	return &ProxyHandler{
		discovery: discovery,
	}
}

// ProxyRequest creates a handler that proxies requests to the specified service
func (h *ProxyHandler) ProxyRequest(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Discover service instance
		instance, err := h.discovery.DiscoverOne(serviceName)
		if err != nil {
			log.Printf("Failed to discover service %s: %v", serviceName, err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("service %s unavailable", serviceName),
			})
			return
		}

		// Build target URL
		target := fmt.Sprintf("http://%s:%d", instance.Address, instance.Port)
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Printf("Failed to parse target URL %s: %v", target, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Customize proxy behavior
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error for %s: %v", serviceName, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"bad gateway"}`))
		}

		// Modify request before proxying
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Preserve original path and query
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

			// Log the proxy request
			log.Printf("Proxying %s %s -> %s", req.Method, c.Request.URL.Path, req.URL.String())
		}

		// Proxy the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// ProxyWithPathRewrite proxies requests with path rewriting
// Example: /api/posts/* -> /* on the posts service
func (h *ProxyHandler) ProxyWithPathRewrite(serviceName, stripPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Discover service instance
		instance, err := h.discovery.DiscoverOne(serviceName)
		if err != nil {
			log.Printf("Failed to discover service %s: %v", serviceName, err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("service %s unavailable", serviceName),
			})
			return
		}

		// Build target URL
		target := fmt.Sprintf("http://%s:%d", instance.Address, instance.Port)
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Printf("Failed to parse target URL %s: %v", target, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error for %s: %v", serviceName, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"bad gateway"}`))
		}

		// Modify request with path rewriting
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

			// Strip prefix if provided
			if stripPrefix != "" {
				req.URL.Path = req.URL.Path[len(stripPrefix):]
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}

			log.Printf("Proxying %s %s -> %s%s",
				req.Method, c.Request.URL.Path, req.URL.Host, req.URL.Path)
		}

		// Proxy the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// Health is the gateway health check handler
func (h *ProxyHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
	})
}
