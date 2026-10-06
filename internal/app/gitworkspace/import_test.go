package gitworkspace

import "testing"

func TestPreviewInfersRepoName(t *testing.T) {
	parentDir := t.TempDir()
	preview, err := Preview(ImportRequest{
		RepoURL:   "https://github.com/YChange01/codex-feishu-link.git",
		ParentDir: parentDir,
	})
	if err != nil {
		t.Fatalf("expected inferred preview, got err=%v", err)
	}
	if preview.DirectoryName != "codex-feishu-link" {
		t.Fatalf("expected inferred repo name, got %q", preview.DirectoryName)
	}
}

func TestPreviewRejectsInvalidCustomName(t *testing.T) {
	if _, err := Preview(ImportRequest{
		RepoURL:       "https://github.com/YChange01/codex-feishu-link.git",
		ParentDir:     t.TempDir(),
		DirectoryName: "bad/name",
	}); err == nil {
		t.Fatal("expected invalid custom directory name to fail")
	}
}
