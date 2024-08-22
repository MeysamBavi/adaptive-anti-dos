package main

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal"
)

func main() {
	internal.RunControlLoop(internal.Config{})
}
