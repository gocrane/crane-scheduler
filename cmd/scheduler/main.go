package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	_ "github.com/gocrane/crane-scheduler/pkg/plugins/apis/config/scheme"

	"github.com/gocrane/crane-scheduler/pkg/plugins/dynamic"
	"github.com/gocrane/crane-scheduler/pkg/plugins/noderesourcetopology"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	cmd := app.NewSchedulerCommand(
		app.WithPlugin(dynamic.Name, dynamic.NewDynamicScheduler),
		app.WithPlugin(noderesourcetopology.Name, noderesourcetopology.New),
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
