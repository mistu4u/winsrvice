// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

type Config struct {
	upgradeWaitTime int
	availxExePath   string
}

var (
	elog debug.Log
	Conf Config
)

type myservice struct{}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	// slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	StateVar := StateStruct{}
	State = &StateVar
	State.set("not running")
	elog.Info(1, "Execute command working")
	config, err := readCofigYaml()
	if err != nil {
		elog.Error(1, fmt.Sprintf("config file could not be read: %v", err))
	}
loop:
	for {
		select {
		case <-tick:
			//Run when the process is not running and hasn't previously errored out
			if State.read() == "not running" {
				if !State.IsFailure {
					State.set("running")
					go run(elog, config["availxExePath"].(string))
				}
			}
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				//forcefully stop the availx thread
				killAvailxAgentInWindows(COBRAEXE)
				//golang.org/x/sys/windows/svc.TestExample is verifying this output.
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				elog.Info(1, testOutput)
				break loop
			// case svc.Pause:
			// 	changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			// 	tick = slowtick
			// case svc.Continue:
			// 	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			// 	tick = fasttick
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func killAvailxAgentInWindows(processName string) {
	elog, err := eventlog.Open(SVCNAME)
	if err != nil {
		fmt.Errorf("debug log could not be opened %w", err.Error())
	}
	//first try to kill gracefully
	kill := exec.Command("TASKKILL", "/IM", processName, "/F", "/T")
	err = kill.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				elog.Info(1, fmt.Sprintf("%v", exitError.ExitCode()))
			}
		}
	}
}

func readCofigYaml() (map[interface{}]interface{}, error) {
	yamlFile, err := ioutil.ReadFile("C://pythian//availxhome//config//winservice.yaml")
	if err != nil {
		elog.Info(1, fmt.Sprintf("yamlFile.Get err   #%v ", err))
	}
	//var c Config
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return data, nil
}
