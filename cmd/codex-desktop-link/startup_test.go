//go:build !windows

package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDesktopStartupReusesLiveDaemon(t *testing.T) {
	s := desktopStarter{probe: func(context.Context) bool { return true }, lock: func(context.Context) (func(), error) {
		t.Fatal("live daemon must not acquire a startup lock")
		return nil, nil
	}, start: func(context.Context) error { t.Fatal("live daemon restarted"); return nil }}
	if err := s.ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopStartupSerializesTwoRobotsAndIDE(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "start.lock")
	var ready atomic.Bool
	var starts atomic.Int32
	s := desktopStarter{
		probe: func(context.Context) bool { return ready.Load() },
		lock:  func(ctx context.Context) (func(), error) { return lockDesktopStartup(ctx, lockPath) },
		start: func(context.Context) error {
			starts.Add(1)
			time.Sleep(20 * time.Millisecond)
			ready.Store(true)
			return nil
		},
		interval: time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.ensure(ctx); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if starts.Load() != 1 {
		t.Fatalf("started daemon %d times", starts.Load())
	}
}

func TestDesktopStartupTimeoutUnlocksForRetry(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "start.lock")
	s := desktopStarter{probe: func(context.Context) bool { return false }, lock: func(ctx context.Context) (func(), error) { return lockDesktopStartup(ctx, lockPath) }, start: func(context.Context) error { return nil }, interval: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := s.ensure(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("startup must have a bounded wait: %v", err)
	}
	retry, cancelRetry := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelRetry()
	unlock, err := lockDesktopStartup(retry, lockPath)
	if err != nil {
		t.Fatalf("timeout leaked startup lock: %v", err)
	}
	unlock()
}
