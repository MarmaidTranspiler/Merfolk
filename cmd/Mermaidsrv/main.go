package main

import (
	"fmt"
	"os"

	"github.com/MarmaidTranspiler/Merfolk/internal/cli"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("no command given, try \"convert\"")
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "convert":
		cli.Convert(args)
	default:
		fmt.Println("unrecognized command")
	}
}
