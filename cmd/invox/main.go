package main

import (
	"os"

	"github.com/0xboris/invox/internal/app"
)

func main() {
	os.Exit(app.Main(os.Args[1:], app.Live()))
}
