//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCancellationInterruptsInheritedBlockingPipes(t *testing.T) {
	connected := make(chan struct{})
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
		close(connected)
		_, _, _ = conn.ReadMessage()
	})
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer inputWriter.Close()
	outputReader, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	defer outputReader.Close()
	// Fd() returns these descriptors to blocking mode, like inherited stdio.
	_ = input.Fd()
	_ = output.Fd()
	in, out, cleanup, err := interruptibleStdio(input, output)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge(ctx, socket, in, out, 1024) }()
	select {
	case <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("connection not established")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("inherited stdio remained blocked")
	}
}
