package main

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config := Config{}
	err := yaml.Unmarshal(cfg, &config)
	if err != nil {
		panic(err)
	}

	for ip, user := range config.Users {
		go applyLoad(ip, user)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
}

//go:embed config.yaml
var cfg []byte

type Config struct {
	Users map[string]User `yaml:"users"`
}

type User struct {
	Start time.Duration `yaml:"start"`
	RPS   float64       `yaml:"rps"`
}

func applyLoad(ip string, user User) {
	time.Sleep(user.Start)
	log.Printf("Starting user %s on with rps %f", ip, user.RPS)
	client := http.Client{Timeout: 3 * time.Second}
	for {
		time.Sleep(time.Second / time.Duration(user.RPS))
		go func() {
			req, err := http.NewRequest(http.MethodGet, "http://localhost:4000/a.png", nil)
			if err != nil {
				log.Println("failed to create request", err)
				return
			}
			req.Header.Add("X-Forwarded-For", ip)
			start := time.Now()
			res, err := client.Do(req)
			if err != nil {
				log.Println("failed to do request", err)
				return
			}
			if res.StatusCode != http.StatusOK {
				log.Println("bad status code", res.StatusCode, ip)
			}
			_, err = io.Copy(io.Discard, res.Body)
			if err != nil {
				log.Println("failed to read body", err)
			}
			d := time.Since(start)
			if d > 1200*time.Millisecond {
				log.Println("took", d)
			}
			err = res.Body.Close()
			if err != nil {
				log.Println("failed to close body", err)
			}
		}()
	}
}
