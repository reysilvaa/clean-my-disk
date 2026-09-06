package main

import (
	"os"

	"clean-my-disk/internal/controller"
)

func main() {
	os.Exit(controller.Run(os.Args[1:], os.Stdout, os.Stderr))
}
