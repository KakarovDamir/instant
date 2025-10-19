// Package gateway implements the API Gateway service logic.
// The gateway handles session validation, service discovery, and request routing
// to backend microservices.
package gateway

import (
	"instant/internal/consul"
	"instant/internal/session"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures and returns the gateway router
func SetupRouter(consulClient *consul.Client, sessionMgr session.Manager) *gin.Engine {
	// Set Gin to release mode for production
	// gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(LoggingMiddleware())
	r.Use(CORSMiddleware())

	// Create proxy handler
	proxyHandler := NewProxyHandler(consulClient)

	// Gateway health check
	r.GET("/health", proxyHandler.Health)

	// Public routes - forward to auth service (no session required)
	auth := r.Group("/auth")
	{
		auth.POST("/request-code", proxyHandler.ProxyWithPathRewrite("auth-service", "/auth"))
		auth.POST("/verify-code", proxyHandler.ProxyWithPathRewrite("auth-service", "/auth"))
		auth.POST("/logout", proxyHandler.ProxyWithPathRewrite("auth-service", "/auth"))
	}

	// Protected routes - require valid session
	api := r.Group("/api")
	api.Use(SessionAuthMiddleware(sessionMgr))
	{
		// Posts service
		// Routes like /api/posts/* -> posts-service/*
		posts := api.Group("/posts")
		{
			posts.Any("/*path", proxyHandler.ProxyWithPathRewrite("posts-service", "/api/posts"))
			posts.Any("", proxyHandler.ProxyRequest("posts-service"))
		}

		// Comments service (when implemented)
		comments := api.Group("/comments")
		{
			comments.Any("/*path", proxyHandler.ProxyWithPathRewrite("comments-service", "/api/comments"))
			comments.Any("", proxyHandler.ProxyRequest("comments-service"))
		}

		// Likes service (when implemented)
		likes := api.Group("/likes")
		{
			likes.Any("/*path", proxyHandler.ProxyWithPathRewrite("likes-service", "/api/likes"))
			likes.Any("", proxyHandler.ProxyRequest("likes-service"))
		}

		// Follow service (when implemented)
		follow := api.Group("/follow")
		{
			follow.Any("/*path", proxyHandler.ProxyWithPathRewrite("follow-service", "/api/follow"))
			follow.Any("", proxyHandler.ProxyRequest("follow-service"))
		}

		// Feed service (when implemented)
		feed := api.Group("/feed")
		{
			feed.Any("/*path", proxyHandler.ProxyWithPathRewrite("feed-service", "/api/feed"))
			feed.Any("", proxyHandler.ProxyRequest("feed-service"))
		}

		// Files service (when implemented)
		files := api.Group("/files")
		{
			files.Any("/*path", proxyHandler.ProxyWithPathRewrite("files-service", "/api/files"))
			files.Any("", proxyHandler.ProxyRequest("files-service"))
		}
	}

	return r
}
