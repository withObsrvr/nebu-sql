package main

import (
	"fmt"
	"os"

	"github.com/withObsrvr/nebu-sql/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
