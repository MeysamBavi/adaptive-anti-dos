package main

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal"
)

func main() {
	cfg := internal.LoadConfig()
	internal.RunControlLoop(cfg)
}
