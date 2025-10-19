package gateway

import (
	"github.com/gin-gonic/gin"
)

// responseWriter wraps gin.ResponseWriter to capture response status and size
type responseWriter struct {
	gin.ResponseWriter
	status int
	size   int
}

// newResponseWriter creates a new response writer wrapper
func newResponseWriter(w gin.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         200, // Default status
		size:           0,
	}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Status returns the captured status code
func (rw *responseWriter) Status() int {
	return rw.status
}

// Size returns the captured response size in bytes
func (rw *responseWriter) Size() int {
	return rw.size
}
