package controller

import (
	"time"
	"log"
)

/**
 * This implements the SheketService, it is Sheket's server.
 */
type SheketController struct {
}

func trace(msg string) func() {
	start := time.Now()
	log.Printf("enter %s", msg)
	return func() { log.Printf("exit %s (%s)", msg, time.Since(start)) }
}
