package main

import (
	"flag"
	"xqdj/internal/audit"
)

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	audit.MustCheckSources(*root)
}
