package main

import (
	"github.com/MeysamBavi/adaptive-anti-dos/server/internal"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

func main() {
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", internal.FileServerHandler)

	addr := ":8080"
	log.Println("Starting file server on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
