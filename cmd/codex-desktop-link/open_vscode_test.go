package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDesktopOpenLaunchesProjectAndVerifiesExactAcknowledgement(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		dir := t.TempDir()
		req := desktopOpenRequest{Version: 1, ID: "request", ThreadID: "thread", CWD: t.TempDir()}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		launches := 0
		err := deliverDesktopOpen(ctx, dir, req, time.Millisecond, time.Millisecond, func(context.Context) error {
			launches++
			raw, e := os.ReadFile(filepath.Join(dir, "request.json"))
			if e != nil {
				t.Fatal(e)
			}
			var written desktopOpenRequest
			if e = json.Unmarshal(raw, &written); e != nil || written.ThreadID != req.ThreadID || written.CWD != req.CWD {
				t.Fatal("lost selected project/thread")
			}
			result := desktopOpenResult{ID: req.ID, ThreadID: req.ThreadID, CWD: req.CWD, Status: "opened"}
			if wrong {
				result.ThreadID = "other-thread"
			}
			data, _ := json.Marshal(result)
			return os.WriteFile(filepath.Join(dir, "request.result.json"), data, 0600)
		})
		cancel()
		if (err != nil) != wrong || launches != 1 {
			t.Fatalf("wrong=%v launches=%d err=%v", wrong, launches, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "request.json")); !os.IsNotExist(err) {
			t.Fatal("request was not cleaned up")
		}
	}
}

func TestDesktopOpenExistingWindowClaimAvoidsNewWindowAndTimesOut(t *testing.T) {
	dir := t.TempDir()
	req := desktopOpenRequest{Version: 1, ID: "existing", ThreadID: "thread", CWD: t.TempDir()}
	if err := os.WriteFile(filepath.Join(dir, "existing.claim"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := deliverDesktopOpen(ctx, dir, req, time.Millisecond, time.Millisecond, func(context.Context) error { t.Fatal("existing project window must be reused"); return nil })
	if err == nil || !strings.Contains(err.Error(), "waiting for VS Code") {
		t.Fatalf("missing visible bounded failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "existing.claim")); !os.IsNotExist(err) {
		t.Fatal("timeout leaked claim")
	}
}
