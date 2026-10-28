package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/sodikinnaa/rdp-generator/internal/api"
	"github.com/sodikinnaa/rdp-generator/internal/service"
)

var (
	version = "1.0.0"
)

func main() {
	port := flag.Int("port", 8787, "Port to listen on")
	host := flag.String("host", "127.0.0.1", "Host address to bind")
	openBrowser := flag.Bool("open", runtime.GOOS == "windows", "Automatically open dashboard in default browser")
	showVersion := flag.Bool("version", false, "Show application version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("RDP Generator v%s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	orch := service.NewOrchestrator()
	server, err := api.NewServer(orch)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize server: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	targetURL := fmt.Sprintf("http://%s", addr)
	fmt.Printf("\n")
	fmt.Printf("============================================================\n")
	fmt.Printf("  RDP Generator - Windows Server on GitHub Codespaces\n")
	fmt.Printf("============================================================\n")
	fmt.Printf("  Web Dashboard: %s\n", targetURL)
	fmt.Printf("  Architecture:  QEMU + KVM inside Codespace via Tailscale\n")
	fmt.Printf("  Press Ctrl+C to stop the local controller.\n")
	fmt.Printf("============================================================\n\n")

	if *openBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openInBrowser(targetURL)
		}()
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	<-stopChan
	fmt.Println("\nShutting down controller server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	fmt.Println("Controller stopped.")
}

func openInBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
