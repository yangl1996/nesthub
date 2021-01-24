package main

import "os/exec"

func openURL(u string) error {
	return exec.Command("open", u).Start()
}
