// +build !windows

package main

import "fmt"

func runningInService(running func()) {
	fmt.Println("Service only for Windows.")
}
