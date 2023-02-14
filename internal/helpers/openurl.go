//go:build !darwin
// +build !darwin

package helpers

import "fmt"

func OpenURL(u string) error {
	fmt.Println("Please open", u, "in browser")
	return nil
}
