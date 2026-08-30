// Package ipc lets a second launch of Pufferfish hand a command to the
// already-running instance instead of starting a duplicate.
package ipc

import (
	"bufio"
	"fmt"
	"net"
)

// addr is the loopback port the running instance listens on.
const addr = "127.0.0.1:52847"

// ShowHistoryCmd asks the running instance to open its history window.
const ShowHistoryCmd = "show-history"

// Listen claims addr for this process and calls handle with every command
// a later launch sends. ok is false when another instance already holds
// addr, in which case the caller is the one that should keep running.
func Listen(handle func(cmd string)) (ln net.Listener, ok bool) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, false
	}

	go serve(ln, handle)
	return ln, true
}

func serve(ln net.Listener, handle func(cmd string)) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go func() {
			defer conn.Close()
			scanner := bufio.NewScanner(conn)
			if scanner.Scan() {
				handle(scanner.Text())
			}
		}()
	}
}

// Send delivers cmd to the running instance. It reports false when no
// instance is listening, so the caller can start one itself.
func Send(cmd string) bool {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	fmt.Fprintln(conn, cmd)
	return true
}
