package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func unixWebSocketServer(t *testing.T, handler func(*websocket.Conn)) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("local desktop Unix transport")
	}
	// Keep the Unix socket below the macOS sockaddr path limit.
	dir, err := os.MkdirTemp("", "cdlink-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(finished)
		conn, err := (&websocket.Upgrader{WriteBufferSize: 16}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		handler(conn)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("server handler did not exit")
		}
	})
	return socket
}

func TestBridgeJSONLinesAndFragmentedWebSocketMessages(t *testing.T) {
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
		for i := 1; i <= 2; i++ {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				t.Error(err)
				return
			}
			var request struct {
				ID int `json:"id"`
			}
			if kind != websocket.TextMessage || json.Unmarshal(data, &request) != nil || request.ID != i {
				t.Errorf("unexpected request %q", data)
				return
			}
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second))
			writer, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.WriteString(writer, "{\n\"id\":")
			_, _ = io.WriteString(writer, string(rune('0'+i))+",\n\"result\":{}}")
			if err := writer.Close(); err != nil {
				t.Error(err)
				return
			}
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		_, _, _ = conn.ReadMessage()
	})
	input, writer := io.Pipe()
	defer writer.Close()
	var output strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge(ctx, socket, input, nopWriteCloser{&output}, 1024) }()
	if _, err := io.WriteString(writer, "\n{\"id\":1}\n{\"id\":2}\n"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "{\"id\":1,\"result\":{}}\n{\"id\":2,\"result\":{}}\n" {
		t.Fatalf("output=%q", got)
	}
}

func TestBridgeRejectsInvalidAndOversizedInput(t *testing.T) {
	for _, input := range []string{"not JSON\n", "[]\n", `{"x":"` + strings.Repeat("a", 256) + `"}` + "\n"} {
		t.Run(input[:min(8, len(input))], func(t *testing.T) {
			socket := unixWebSocketServer(t, func(conn *websocket.Conn) { _, _, _ = conn.ReadMessage() })
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := bridge(ctx, socket, io.NopCloser(strings.NewReader(input)), nopWriteCloser{io.Discard}, 128)
			if err == nil || !strings.Contains(err.Error(), "stdin") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestBridgeRejectsInvalidAndOversizedServerFrames(t *testing.T) {
	for _, test := range []struct {
		name    string
		kind    int
		message string
	}{
		{"binary", websocket.BinaryMessage, `{}`},
		{"invalid JSON", websocket.TextMessage, "not JSON"},
		{"oversized", websocket.TextMessage, `{"x":"` + strings.Repeat("a", 256) + `"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
				_ = conn.WriteMessage(test.kind, []byte(test.message))
				_, _, _ = conn.ReadMessage()
			})
			input, writer := io.Pipe()
			defer writer.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := bridge(ctx, socket, input, nopWriteCloser{io.Discard}, 128)
			if err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestBridgeCancellationClosesBlockedStdinAndStdout(t *testing.T) {
	sent := make(chan struct{})
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"result":{}}`))
		close(sent)
		_, _, _ = conn.ReadMessage()
	})
	input, writer := io.Pipe()
	defer writer.Close()
	reader, output := io.Pipe()
	defer reader.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge(ctx, socket, input, output, 1024) }()
	select {
	case <-sent:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not send")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("bridge remained blocked after cancellation")
	}
}

func TestBridgeStdinEOFOnlyClosesItsConnection(t *testing.T) {
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
		_, _, err := conn.ReadMessage()
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			t.Errorf("unexpected close: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := bridge(ctx, socket, io.NopCloser(strings.NewReader("")), nopWriteCloser{io.Discard}, 1024); err != nil {
		t.Fatal(err)
	}
}

func TestBridgeUnexpectedDisconnectUnblocksStdin(t *testing.T) {
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) { _ = conn.Close() })
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := bridge(ctx, socket, input, nopWriteCloser{io.Discard}, 1024)
	if err == nil || !strings.Contains(err.Error(), "read desktop WebSocket") {
		t.Fatalf("err=%v", err)
	}
}

type failingOutput struct{}

func (failingOutput) Write([]byte) (int, error) { return 0, errors.New("downstream closed") }
func (failingOutput) Close() error              { return nil }

func TestBridgeOutputFailureUnblocksStdin(t *testing.T) {
	socket := unixWebSocketServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":1}`))
		_, _, _ = conn.ReadMessage()
	})
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := bridge(ctx, socket, input, failingOutput{}, 1024)
	if err == nil || !strings.Contains(err.Error(), "downstream closed") {
		t.Fatalf("err=%v", err)
	}
}
