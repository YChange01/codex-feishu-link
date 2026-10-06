package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	maxMessageBytes = 64 * 1024 * 1024
	writeTimeout    = 15 * time.Second
)

// bridge owns the supplied stream handles until it returns. Closing all three
// streams before joining the pumps unblocks both an idle stdin and a blocked
// downstream writer; shutting down this connection never stops the daemon.
func bridge(ctx context.Context, socket string, stdin io.ReadCloser, stdout io.WriteCloser, limit int) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	conn, response, err := dialer.DialContext(ctx, "ws://localhost/", nil)
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("connect to desktop Codex daemon: %w", err)
	}
	conn.SetReadLimit(int64(limit))
	results := make(chan error, 2)
	go func() { results <- jsonLinesToWebSocket(conn, stdin, limit) }()
	go func() { results <- webSocketToJSONLines(conn, stdout) }()

	var result error
	remaining := 2
	select {
	case result = <-results:
		remaining--
	case <-ctx.Done():
		result = ctx.Err()
	}
	if result == nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	}
	_ = conn.Close()
	_ = stdin.Close()
	_ = stdout.Close()
	for ; remaining > 0; remaining-- {
		<-results
	}
	return result
}

func jsonLinesToWebSocket(conn *websocket.Conn, input io.Reader, limit int) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, min(64*1024, limit+1)), limit+1)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if len(line) > limit {
			return fmt.Errorf("stdin JSONL message exceeds %d bytes", limit)
		}
		if !validJSONObject(line) {
			return errors.New("stdin must contain one valid JSON object per line")
		}
		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, line); err != nil {
			return fmt.Errorf("write desktop WebSocket: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stdin JSONL (maximum %d bytes per message): %w", limit, err)
	}
	return nil
}

func webSocketToJSONLines(conn *websocket.Conn, output io.Writer) error {
	for {
		kind, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return fmt.Errorf("read desktop WebSocket: %w", err)
		}
		if kind != websocket.TextMessage || !validJSONObject(bytes.TrimSpace(message)) {
			return errors.New("desktop WebSocket sent a non-text or invalid JSON object")
		}
		var line bytes.Buffer
		if err := json.Compact(&line, message); err != nil {
			return fmt.Errorf("encode desktop response: %w", err)
		}
		line.WriteByte('\n')
		if _, err := io.Copy(output, &line); err != nil {
			return fmt.Errorf("write stdout JSONL: %w", err)
		}
	}
}

func validJSONObject(message []byte) bool {
	return len(message) > 0 && message[0] == '{' && utf8.Valid(message) && json.Valid(message)
}
