package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/component-base/version"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane-scheduler/cmd/controller/app/config"
	"github.com/gocrane/crane-scheduler/cmd/controller/app/options"
	"github.com/gocrane/crane-scheduler/pkg/controller/annotator"
)

// NewControllerCommand creates a *cobra.Command object with default parameters
func NewControllerCommand() *cobra.Command {
	o, err := options.NewOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	cmd := &cobra.Command{
		Use: "crane-scheduler-controller",
		Long: `The Crane Scheduler Controller is a kubernetes controller, which is used for annotating
		nodes with real load imformation sourced from Prometheus defaultly. `,
		Run: func(cmd *cobra.Command, args []string) {

			c, err := o.Config()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			stopCh := make(chan struct{})
			if err := Run(c.Complete(), stopCh); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	err = o.Flags(cmd.Flags())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}

	return cmd
}

// Run executes controller based on the given configuration.
func Run(cc *config.CompletedConfig, stopCh <-chan struct{}) error {

	klog.Infof("Starting Controller version %+v", version.Get())

	run := func(ctx context.Context) {
		annotatorController := annotator.NewNodeAnnotator(
			cc.KubeInformerFactory.Core().V1().Nodes(),
			cc.KubeInformerFactory.Core().V1().Events(),
			cc.KubeClient,
			cc.PromClient,
			*cc.Policy,
			cc.AnnotatorConfig.BindingHeapSize,
		)

		cc.KubeInformerFactory.Start(stopCh)

		panic(annotatorController.Run(int(cc.AnnotatorConfig.ConcurrentSyncs), stopCh))
	}

	if !cc.LeaderElection.LeaderElect {
		run(context.TODO())
		panic("unreachable")
	}

	id, err := os.Hostname()
	if err != nil {
		return err
	}

	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id = id + "_" + string(uuid.NewUUID())
	rl, err := resourcelock.New(cc.LeaderElection.ResourceLock,
		cc.LeaderElection.ResourceNamespace,
		cc.LeaderElection.ResourceName,
		cc.LeaderElectionClient.CoreV1(),
		cc.LeaderElectionClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: cc.EventRecorder,
		})
	if err != nil {
		panic(err)
	}

	electionChecker := leaderelection.NewLeaderHealthzAdaptor(time.Second * 20)
	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: cc.LeaderElection.LeaseDuration.Duration,
		RenewDeadline: cc.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:   cc.LeaderElection.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				panic("leaderelection lost")
			},
		},
		WatchDog: electionChecker,
		Name:     "crane-scheduler-controller",
	})

	panic("unreachable")
}
