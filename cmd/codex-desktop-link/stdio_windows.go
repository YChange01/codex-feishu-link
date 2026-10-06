//go:build windows

package main

import (
	"fmt"
	"io"
)

func interruptibleStdio(io.ReadCloser, io.WriteCloser) (io.ReadCloser, io.WriteCloser, func(), error) {
	return nil, nil, nil, fmt.Errorf("this desktop launcher is for the local Unix installation")
}
