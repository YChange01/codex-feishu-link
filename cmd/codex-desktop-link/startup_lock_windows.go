//go:build windows

package main

import (
	"context"
	"fmt"
)

func lockDesktopStartup(context.Context, string) (func(), error) {
	return nil, fmt.Errorf("automatic shared desktop startup is supported on Unix only")
}
