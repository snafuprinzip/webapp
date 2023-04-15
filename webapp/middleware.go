package webapp

import "net/http"

// Middleware is a chain of sequential http Handlers
type Middleware []http.Handler

// MiddlewareResponseWriter is a specialized Version of the http.ResponseWriter that marks if
// a Handler in the middleware chain has written to it
type MiddlewareResponseWriter struct {
	http.ResponseWriter
	written bool
}

// Add appends a new Handler at the end of the middleware chain
func (m *Middleware) Add(handler http.Handler) {
	*m = append(*m, handler)
}

// ServeHTTP loops through all Handlers in the middleware chain and serves each
// until one writes a response (with the Write or WriteHeader functions below)
// or until we answer with a 404 NotFound
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Wrap the supplied ResponseWriter
	mw := NewMiddlewareResponseWriter(w)

	// Loop through all of the registered handlers
	for _, handler := range m {
		// Call the handler with our MiddlewareResponseWriter
		handler.ServeHTTP(mw, r)

		// If there was a write, stop processing
		if mw.written {
			return
		}
	}
	// If no handlers wrote to the response, it’s a 404
	http.NotFound(w, r)
}

// NewMiddlewareResponseWriter creates a new MiddlewareResponseWriter instance
func NewMiddlewareResponseWriter(w http.ResponseWriter) *MiddlewareResponseWriter {
	return &MiddlewareResponseWriter{
		ResponseWriter: w,
	}
}

// Write writes into the MiddlewareResponseWriter and returns the number of bytes written
func (w *MiddlewareResponseWriter) Write(bytes []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(bytes)
}

// WriteHeader writes a return code into the header of the MiddlewareResponseWriter
func (w *MiddlewareResponseWriter) WriteHeader(code int) {
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}
