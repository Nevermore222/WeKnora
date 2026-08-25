package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWaitForDesktopBackendWaitsForReadySignal(t *testing.T) {
	ready := make(chan string, 1)
	startupErr := make(chan error, 1)

	go func() {
		time.Sleep(20 * time.Millisecond)
		ready <- "http://127.0.0.1:52701"
	}()

	got, err := waitForDesktopBackend(ready, startupErr, time.Second)
	if err != nil {
		t.Fatalf("waitForDesktopBackend returned error: %v", err)
	}
	if got != "http://127.0.0.1:52701" {
		t.Fatalf("expected ready backend URL, got %q", got)
	}
}

func TestWaitForDesktopBackendReturnsStartupFailure(t *testing.T) {
	ready := make(chan string, 1)
	startupErr := make(chan error, 1)
	startupErr <- errors.New("database initialization failed")

	_, err := waitForDesktopBackend(ready, startupErr, time.Second)
	if err == nil {
		t.Fatal("expected backend startup error")
	}
	if !strings.Contains(err.Error(), "database initialization failed") {
		t.Fatalf("expected startup error to be preserved, got %v", err)
	}
}
