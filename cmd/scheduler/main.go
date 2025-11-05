package main

import (
	"log"
	"time"

	"github.com/dhima/event-trigger-platform/internal/scheduler"
)

func main() {
	engine := scheduler.NewEngine(time.Second * 5)
	log.Fatal(engine.Run())
}
