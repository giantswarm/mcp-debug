package main

import (
	"github.com/giantswarm/mcp-debug/cmd"
	"github.com/giantswarm/mcp-debug/pkg/project"
)

func main() {
	cmd.SetVersion(project.Version())
	cmd.Execute()
}
