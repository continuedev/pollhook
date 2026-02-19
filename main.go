package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var version = "dev"

func main() {
	log.SetFlags(log.Ltime)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		runServeCmd(os.Args[2:])
	case "test":
		runTestCmd(os.Args[2:])
	case "version":
		fmt.Printf("pollhook %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: pollhook <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  serve     Run pollers and deliver webhooks")
	fmt.Fprintln(os.Stderr, "  test      Validate config, run commands, show extracted items (dry run)")
	fmt.Fprintln(os.Stderr, "  version   Print version")
}

func runServeCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "pollhook.yaml", "Path to config file")
	stateDir := fs.String("state-dir", defaultStateDir(), "Directory for state persistence")
	fs.Parse(args)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	state := NewState(*stateDir)
	if err := state.Load(); err != nil {
		log.Fatalf("state load error: %v", err)
	}

	log.Printf("pollhook %s starting with %d source(s)", version, len(cfg.Sources))

	ctx, cancel := context.WithCancel(context.Background())

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received %s, shutting down...", sig)
		cancel()
	}()

	runServe(ctx, cfg, state)
	log.Println("pollhook stopped")
}

func runTestCmd(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	configPath := fs.String("config", "pollhook.yaml", "Path to config file")
	fs.Parse(args)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	fmt.Printf("Config loaded: %d source(s)\n\n", len(cfg.Sources))
	if err := runTest(cfg); err != nil {
		log.Fatalf("test error: %v", err)
	}
}

func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pollhook"
	}
	return filepath.Join(home, ".pollhook")
}
