package main

import (
	"log"
	"os"

	"github.com/YChange01/codex-feishu-link/testkit/mockclaude"
)

func main() {
	if err := mockclaude.RunIO(mockclaude.NewFromEnvAndArgs(os.Args[1:]), os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
