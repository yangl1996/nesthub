//go:build !darwin
// +build !darwin

package main

import "fmt"

func openURL(u string) error {
	fmt.Println("Please open", u, "in browser")
	return nil
}
