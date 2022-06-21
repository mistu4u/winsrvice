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
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

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

func run(elog debug.Log, path string, waitTime int) {
	State.IsFailure = false
	cmd := exec.Command(path, "run")
	stdout, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				if exitError.ExitCode() == 2 {
					//availx agent got upgraded
					State.IsFailure = false
					elog.Info(1, fmt.Sprintf("exe upgraded, going to wait for %d minutes to start it", waitTime))
					time.Sleep(time.Duration(waitTime) * time.Second)
					runService(SVCNAME, true)
				}
				//update the service status to stopped
				err = controlService(SVCNAME, svc.Stop, svc.Stopped)
				elog.Error(1, fmt.Sprintf("error in stopping service: %v", err))
				State.IsFailure = true
				elog.Error(1, fmt.Sprintf("fatal occurred : %v", exitError))
			}
		}
	}
	elog.Info(1, string(stdout))
	State.set("not running")
}
