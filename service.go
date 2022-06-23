// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import (
	"fmt"
	"github.com/shirou/gopsutil/process"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
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
			//elog.Info(1, fmt.Sprintf("pid inside tick %v", Pid))
			if State.read() == "not running" {
				if !State.IsFailure {
					State.set("running")
					elog.Info(1, "inside tick, running")
					go run(elog, config[AVAILX_EXE_PATH].(string), config[UPGRADE_WAIT_TIME].(int))
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
				elog.Info(1, fmt.Sprintf("pid before stop %v", Pid))
				killAvailxAgentInWindows(Pid)
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

func runAgent(name string, isDebug bool) {
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

func killAvailxAgentInWindows(lpid int32) {
	//conf, _ := readCofigYaml()
	//fmt.Println(conf["availxExePath"])
	elog, _ = eventlog.Open(SVCNAME)
	defer elog.Close()
	//file, err := ioutil.ReadFile(conf["availxExePath"].(string) + "msw.lock")
	//if err != nil {
	//	elog.Error(1, fmt.Sprintf("Pid value was not retrieved from the file %v", err.Error()))
	//}
	//i, _ := strconv.Atoi(string(file))
	//Pid = int32(i)
	elog.Info(1, fmt.Sprintf("pid value received in kill is %v", lpid))
	if lpid == 0 {
		elog.Error(1, "Pid value can not be zero")
		return
	} else {
		elog.Info(1, fmt.Sprintf("pid value received is %v", lpid))
		p, err := process.NewProcess(lpid)
		if err != nil {
			elog.Error(1, fmt.Sprintf("error while getting process details %v", err.Error()))
		}
		err = p.Terminate()
		if err != nil {
			elog.Error(1, fmt.Sprintf("error while killing the agent %v", err.Error()))
		}
		//check if the PID still exists
		//Check for 30 seconds if the service is still running or not
		currTime := time.Now()
		stopTime := time.Now().Add(11 * time.Second)
		var exists bool
		for currTime.Before(stopTime) {
			exists, err = process.PidExists(lpid)
			if err != nil {
				elog.Error(1, fmt.Sprintf("could not check if process still exists %v", err.Error()))
				break
			}
			if !exists {
				break
			}
			time.Sleep(1 * time.Second)
			currTime = time.Now()
		}
		if exists {
			err := p.Kill()
			if err != nil {
				elog.Error(1, fmt.Sprintf("process could not be force killed %v", err.Error()))
			} else {
				elog.Info(1, fmt.Sprintf("process was force killed %v", err.Error()))
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
