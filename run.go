// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

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
	//Get the pid of the agent
	pid := cmd.Process.Pid
	elog.Info(1, fmt.Sprintf("after start, pid %v created", pid))
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				if exitError.ExitCode() == 2 {
					//availx agent got upgraded
					State.IsFailure = false
					elog.Info(1, fmt.Sprintf("exe upgraded, going to wait for %d seconds to start it", waitTime))
					time.Sleep(time.Duration(waitTime) * time.Second)
					//No need to start the service, Windows is going to start it again after 1 minute, set the
					//recovery option in services
					//runAgent(SVCNAME, true)
					elog.Info(1, fmt.Sprintf("wait ended, exiting with code %d", 2))
					os.Exit(2)
				}
				//no need to update the status to stopped, windows won't try to restart since it is a genuine error
				//err = controlService(SVCNAME, svc.Stop, svc.Stopped)
				//elog.Error(1, fmt.Sprintf("error in stopping service: %v", err))
				State.IsFailure = true
				elog.Error(1, fmt.Sprintf("fatal occurred : %v", exitError))
				os.Exit(0)
			}
		}
	}
	elog.Info(1, string(stdout))
	State.set("not running")
}
