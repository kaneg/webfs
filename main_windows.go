// +build windows

package main

import (
	"os/exec"
	"golang.org/x/sys/windows/svc"
	"time"
	"fmt"
	"regexp"
	"strings"
)

type WindowsService struct {
	running func()
}

func splitCommands(command string) []string {
	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	m := r.FindAllString(command, -1)
	for i, item := range m {
		if strings.HasPrefix(item, `"`) {
			item = item[1 : len(item)-1]
		}
		if strings.HasSuffix(item, `"`) {
			item = item[:len(item)-1]
		}
		m[i] = item
	}
	return m
}

func getStartCommands(command string) *exec.Cmd {
	command = "/c " + command
	args := splitCommands(command)
	fmt.Println("args:", args)
	return exec.Command("cmd", args...)
}

func (m *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	go m.running()
loop:
	for {
		select {
		case <-tick:
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:

			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runningInService(running func()) {
	svc.Run("WebFS", &WindowsService{running})
}
