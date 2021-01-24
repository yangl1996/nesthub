// +build !darwin

package main

import "fmt"

func openURL(u string) error {
	fmt.Println("Please open", authURL, "in browser")
	return nil
}
