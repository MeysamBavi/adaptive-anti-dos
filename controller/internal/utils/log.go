package utils

import (
	"log"
	"os"
)

func GetLogger(prefix string) *log.Logger {
	return log.New(os.Stderr, prefix, log.Ltime|log.Ldate)
}
