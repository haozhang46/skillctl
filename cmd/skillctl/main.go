package main

import (
	"os"

	"github.com/hz/skillctl/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
