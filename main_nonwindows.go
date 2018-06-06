// +build !windows

package main

import "fmt"
import "os/exec"

func runningInService(running func()) {
	fmt.Println("Service only for Windows.")
}

func getStartCommands(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
