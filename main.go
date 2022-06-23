// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

// Example service program that beeps.
//
// The program demonstrates how to create Windows service and
// install / remove it on a computer. It also shows how to
// stop / start / pause / continue any service, and how to
// write to event log. It also shows how to use debug
// facilities available in debug package.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	conf, _ := readCofigYaml()
	fmt.Println(conf[AVAILX_EXE_PATH])
	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}
	if inService {
		runAgent(SVCNAME, false)
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		runAgent(SVCNAME, true)
		return
	case "install":
		err = installService(SVCNAME, SVC_DESC)
	case "remove":
		err = removeService(SVCNAME)
	case "start":
		err = startService(SVCNAME)
		//Stopping the service from the cli is disabled
	/*case "stop":
	err = controlService(svcName, svc.Stop, svc.Stopped)
	killAvailxAgentInWindows(true)
	case "pause":
		err = controlService(SVCNAME, svc.Pause, svc.Paused)
	case "continue":
		err = controlService(SVCNAME, svc.Continue, svc.Running)
	/*case "restart":
	err = restartService(svcName, svc.Stop, svc.Stopped)*/
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, SVCNAME, err)
	}
}
