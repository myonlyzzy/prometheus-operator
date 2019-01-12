package main

import (
	"fmt"
	"github.com/myonlyzzy/prometheus-operator/cmd/prometheus-operator/app"
	"k8s.io/apiserver/pkg/util/logs"
	"os"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	command := app.NewControllerManagerCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
