//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func interruptibleStdio(input io.ReadCloser, output io.WriteCloser) (io.ReadCloser, io.WriteCloser, func(), error) {
	cleanInput := func() {}
	if file, ok := input.(*os.File); ok {
		var err error
		input, cleanInput, err = interruptibleFile(file)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	cleanOutput := func() {}
	if file, ok := output.(*os.File); ok {
		var err error
		output, cleanOutput, err = interruptibleFile(file)
		if err != nil {
			cleanInput()
			return nil, nil, nil, err
		}
	}
	return input, output, func() { cleanInput(); cleanOutput() }, nil
}

// Inherited stdio pipes can be blocking descriptors that os.File.Close cannot
// interrupt. A nonblocking duplicate lets Go's poller cancel pending reads and
// writes. Restore the original flags after the pumps have stopped.
func interruptibleFile(file *os.File) (*os.File, func(), error) {
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if info.Mode().IsRegular() {
		return file, func() {}, nil
	}
	if info.Mode()&os.ModeNamedPipe == 0 && info.Mode()&os.ModeSocket == 0 {
		return nil, nil, fmt.Errorf("shared desktop app-server requires piped stdin/stdout, not a terminal")
	}
	originalFD := int(file.Fd())
	flags, err := unix.FcntlInt(uintptr(originalFD), unix.F_GETFL, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect stdio pipe: %w", err)
	}
	fd, err := unix.Dup(originalFD)
	if err != nil {
		return nil, nil, fmt.Errorf("duplicate stdio pipe: %w", err)
	}
	unix.CloseOnExec(fd)
	if err := unix.SetNonblock(fd, true); err != nil {
		_ = unix.Close(fd)
		return nil, nil, fmt.Errorf("prepare interruptible stdio pipe: %w", err)
	}
	pollable := os.NewFile(uintptr(fd), file.Name())
	return pollable, func() {
		_ = pollable.Close()
		if flags&unix.O_NONBLOCK == 0 {
			_ = unix.SetNonblock(originalFD, false)
		}
	}, nil
}
