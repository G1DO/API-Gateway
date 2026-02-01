package middleware

import "net/http"

// ResponseCapture wraps http.ResponseWriter to capture the status code
// and bytes written. Needed by logging and metrics middleware since
// http.ResponseWriter doesn't expose the status after WriteHeader().
type ResponseCapture struct {
	http.ResponseWriter
	StatusCode int
	Written    int64
}

// NewResponseCapture wraps a ResponseWriter.
func NewResponseCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{
		ResponseWriter: w,
		StatusCode:     http.StatusOK, // default if WriteHeader is never called
	}
}

// WriteHeader captures the status code then delegates.
func (rc *ResponseCapture) WriteHeader(code int) {
	rc.StatusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written then delegates.
func (rc *ResponseCapture) Write(b []byte) (int, error) {
	n, err := rc.ResponseWriter.Write(b)
	rc.Written += int64(n)
	return n, err
}
