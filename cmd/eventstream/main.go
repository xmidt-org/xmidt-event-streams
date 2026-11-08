// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/xmidt-org/xmidt-event-streams/internal/install"
)

func main() {
	os.Exit(run())
}

func run() int {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	err := install.EventStream(os.Args[1:], true)
	if err == nil {
		return 0
	}

	fmt.Fprintln(os.Stderr, err)
	return -1
}
