package helpers

import (
	"log"
	"os/exec"
)

func OpenURL(u string) error {
	log.Printf("Opening %s in browser", u)
	return exec.Command("open", u).Start()
}
