package main

import (
	"fmt"
	"os"

	"github.com/Gogoro/marg.sh/internal/marg"
)

func main() {
	if err := marg.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "marg:", err)
		os.Exit(1)
	}
}
