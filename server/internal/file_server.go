package internal

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	filesDirectory = "/files"
)

func FileServerHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	wrw := NewWrappedResponseWriter(w)
	w = wrw
	fileName := strings.TrimPrefix(r.URL.Path, "/")

	ip := r.RemoteAddr
	if ips := strings.Split(ip, ":"); len(ips) > 1 {
		ip = ips[0]
	}

	defer func() {
		ObserveRequestMetrics(start, wrw.Status(), ip)
	}()

	http.ServeFile(w, r, filepath.Join(filesDirectory, fileName))
}
