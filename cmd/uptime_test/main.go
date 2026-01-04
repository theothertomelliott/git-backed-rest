package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/theothertomelliott/git-backed-rest/client"
	"github.com/tjarratt/babble"
)

func main() {
	// Parse command line flags
	repetitionsPerMinute := flag.Int("repetitions", 1, "Number of repetitions per minute")
	duration := flag.Duration("duration", time.Hour, "How long to run the test")
	flag.Parse()

	if *repetitionsPerMinute < 1 {
		log.Fatalf("repetitions must be at least 1")
	}

	log.Printf("Starting uptime test: %d repetitions per minute for %v", *repetitionsPerMinute, *duration)

	// Calculate interval between repetitions
	interval := time.Minute / time.Duration(*repetitionsPerMinute)
	log.Printf("Actions will repeat every %v", interval)

	// Create context for the entire test
	ctx := context.Background()

	// Run the uptime test
	runUptimeTest(ctx, interval, *duration)
}

func runUptimeTest(ctx context.Context, interval time.Duration, duration time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Create a ticker for the overall duration
	durationTicker := time.NewTicker(duration)
	defer durationTicker.Stop()

	actionCount := 0

	// Execute first action immediately
	actionCount++
	log.Printf("=== Action #%d at %v ===", actionCount, time.Now().Format("15:04:05"))

	if err := executeUserActions(ctx); err != nil {
		log.Printf("Action #%d failed: %v", actionCount, err)
	} else {
		log.Printf("Action #%d completed successfully", actionCount)
	}

	for {
		select {
		case <-durationTicker.C:
			log.Printf("Test duration reached. Total actions: %d", actionCount)
			return
		case <-ticker.C:
			actionCount++
			log.Printf("=== Action #%d at %v ===", actionCount, time.Now().Format("15:04:05"))

			if err := executeUserActions(ctx); err != nil {
				log.Printf("Action #%d failed: %v", actionCount, err)
			} else {
				log.Printf("Action #%d completed successfully", actionCount)
			}
		}
	}
}

func executeUserActions(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// Create client for this action sequence
	c := client.New("http://localhost:8080")

	// Generate random names for resources
	resource1 := generateRandomResourceName()
	resource2 := generateRandomResourceName()

	log.Printf("Creating first resource: %s", resource1)
	if err := c.POST(ctx, resource1, []byte(`{"type": "first", "timestamp": "`+time.Now().Format(time.RFC3339)+`"}`)); err != nil {
		return fmt.Errorf("failed to create first resource: %w", err)
	}

	log.Printf("Reading first resource: %s", resource1)
	result, err := c.GET(ctx, resource1)
	if err != nil {
		return fmt.Errorf("failed to read first resource: %w", err)
	}
	log.Printf("✓ First resource content: %s", string(result))

	log.Printf("Updating first resource: %s", resource1)
	if err := c.PUT(ctx, resource1, []byte(`{"type": "first", "updated": true, "timestamp": "`+time.Now().Format(time.RFC3339)+`"}`)); err != nil {
		return fmt.Errorf("failed to update first resource: %w", err)
	}

	log.Printf("Creating second resource: %s", resource2)
	if err := c.POST(ctx, resource2, []byte(`{"type": "second", "timestamp": "`+time.Now().Format(time.RFC3339)+`"}`)); err != nil {
		return fmt.Errorf("failed to create second resource: %w", err)
	}

	log.Printf("Reading second resource: %s", resource2)
	result, err = c.GET(ctx, resource2)
	if err != nil {
		return fmt.Errorf("failed to read second resource: %w", err)
	}
	log.Printf("✓ Second resource content: %s", string(result))

	log.Printf("Deleting second resource: %s", resource2)
	if err := c.DELETE(ctx, resource2); err != nil {
		return fmt.Errorf("failed to delete second resource: %w", err)
	}

	log.Printf("✓ All user actions completed successfully")
	return nil
}

func generateRandomResourceName() string {
	// Use babble library for consistent random name generation like create_repo
	babbler := babble.NewBabbler()
	babbler.Count = 3
	babbler.Separator = "-"
	return "/" + babbler.Babble()
}
