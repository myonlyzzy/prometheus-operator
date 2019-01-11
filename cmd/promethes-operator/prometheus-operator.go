package main

import (
	"fmt"
	"github.com/myonlyzzy/prometheus-operator/cmd/promethes-operator/app"
	"os"
)

func main() {
	command := app.NewControllerManagerCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
