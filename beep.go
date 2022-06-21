// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

// BUG(brainman): MessageBeep Windows api is broken on Windows 7,
// so this example does not beep when runs as service on Windows 7.

var (
	beepFunc = syscall.MustLoadDLL("user32.dll").MustFindProc("MessageBeep")
)

func beep() {
	beepFunc.Call(0xffffffff)
}

type StateStruct struct {
	sync.Mutex
	Desc      string
	IsFailure bool
}

var State *StateStruct

func (s *StateStruct) set(desc string) {
	s.Lock()
	defer s.Unlock()
	s.Desc = desc
}
func (s *StateStruct) read() string {
	s.Lock()
	defer s.Unlock()
	return s.Desc
}

func run(elog debug.Log, path string) {
	State.IsFailure = false
	cmd := exec.Command(path, "run")
	stdout, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				//update the service status to stopped
				err = controlService(SVCNAME, svc.Stop, svc.Stopped)
				elog.Info(1, fmt.Sprintf("error in stopping service: %v", err))
				State.IsFailure = true
				elog.Info(1, fmt.Sprintf("%v", exitError))
			}
		}
	}
	elog.Info(1, string(stdout))
	State.set("not running")
}
