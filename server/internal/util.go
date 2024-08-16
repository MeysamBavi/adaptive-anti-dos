package internal

import (
	"net/http"
)

type WrappedResponseWriter struct {
	http.ResponseWriter
	status int
}

func NewWrappedResponseWriter(w http.ResponseWriter) *WrappedResponseWriter {
	return &WrappedResponseWriter{w, http.StatusOK}
}

func (w *WrappedResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *WrappedResponseWriter) Status() int {
	return w.status
}
