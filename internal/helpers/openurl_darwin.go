package helpers

import "os/exec"

func OpenURL(u string) error {
	return exec.Command("open", u).Start()
}
