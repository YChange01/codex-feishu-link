package main

import (
	"os"

	"github.com/YChange01/codex-feishu-link/internal/app/launcher"
)

var version = "dev"
var branch = "dev"

func main() {
	args := append([]string{"wrapper"}, os.Args[1:]...)
	os.Exit(launcher.Main(launcher.Options{
		Args:    args,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: version,
		Branch:  branch,
	}))
}
