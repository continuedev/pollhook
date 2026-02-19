package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

// execCommand runs a command via sh -c with a 60s timeout.
func execCommand(command string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timed out after 60s")
		}
		return nil, fmt.Errorf("command failed: %w", err)
	}
	return out, nil
}

// pollOnce runs one poll cycle for a source: exec → extract → dedup → deliver.
// Returns the number of new items delivered.
func pollOnce(src Source, state *State) int {
	out, err := execCommand(src.Command)
	if err != nil {
		log.Printf("[%s] %v", src.Name, err)
		return 0
	}

	items, err := ExtractItems(out, src.Items)
	if err != nil {
		log.Printf("[%s] extract items: %v", src.Name, err)
		return 0
	}

	delivered := 0
	now := time.Now().UTC().Format(time.RFC3339)

	for _, item := range items {
		id, err := ExtractID(item, src.ID)
		if err != nil {
			log.Printf("[%s] extract id: %v", src.Name, err)
			continue
		}

		if state.HasID(src.Name, id) {
			continue
		}

		payload := WebhookPayload{
			Source:   src.Name,
			Item:     item,
			PolledAt: now,
		}

		if err := DeliverWebhook(src.Webhook.URL, src.Webhook.Secret, payload); err != nil {
			log.Printf("[%s] deliver %s: %v", src.Name, id, err)
			continue
		}

		state.AddID(src.Name, id)
		delivered++
		log.Printf("[%s] delivered %s", src.Name, id)
	}

	return delivered
}

// runServe starts goroutine-per-source poll loops with periodic state flushing.
func runServe(ctx context.Context, cfg *Config, state *State) {
	var wg sync.WaitGroup

	for _, src := range cfg.Sources {
		src := src
		wg.Add(1)
		go func() {
			defer wg.Done()

			log.Printf("[%s] polling every %s", src.Name, src.Interval.Duration)

			// Immediate poll on startup
			pollOnce(src, state)

			ticker := time.NewTicker(src.Interval.Duration)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pollOnce(src, state)
				}
			}
		}()
	}

	// Periodic state flush every 30s
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := state.Save(); err != nil {
					log.Printf("state save: %v", err)
				}
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()

	// Final state flush on shutdown
	if err := state.Save(); err != nil {
		log.Printf("final state save: %v", err)
	}
}

// runTest validates config, runs each command once, and shows extracted items.
func runTest(cfg *Config) error {
	for _, src := range cfg.Sources {
		fmt.Printf("=== %s ===\n", src.Name)
		fmt.Printf("Command: %s\n", src.Command)
		fmt.Printf("Interval: %s\n", src.Interval.Duration)
		fmt.Printf("Webhook: %s\n", src.Webhook.URL)
		fmt.Println()

		out, err := execCommand(src.Command)
		if err != nil {
			fmt.Printf("ERROR running command: %v\n\n", err)
			continue
		}
		fmt.Printf("Raw output: %d bytes\n", len(out))

		items, err := ExtractItems(out, src.Items)
		if err != nil {
			fmt.Printf("ERROR extracting items: %v\n\n", err)
			continue
		}
		fmt.Printf("Items found: %d\n\n", len(items))

		for i, item := range items {
			if i >= 5 {
				fmt.Printf("... and %d more items\n", len(items)-5)
				break
			}

			id, err := ExtractID(item, src.ID)
			if err != nil {
				fmt.Printf("  [%d] ERROR extracting id: %v\n", i, err)
				continue
			}

			payload := WebhookPayload{
				Source:   src.Name,
				Item:     item,
				PolledAt: time.Now().UTC().Format(time.RFC3339),
			}
			payloadJSON, _ := json.MarshalIndent(payload, "  ", "  ")
			fmt.Printf("  [%d] id=%s\n  Payload:\n  %s\n\n", i, id, payloadJSON)
		}
	}
	return nil
}
