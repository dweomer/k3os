package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/rancher/k3os/pkg/apis/k3os.io/v1"
	"github.com/rancher/k3os/pkg/controller"
	"github.com/rancher/k3os/pkg/generated/controllers/k3os.io"
	"github.com/rancher/k3os/pkg/system"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/batch"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Command is the `agent` sub-command, it is the k3OS resource controller.
func Command() cli.Command {
	return cli.Command{
		Name:  "agent",
		Usage: "custom resource(s) controller",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "threads",
				EnvVar: "K3OS_OPERATOR_THREADS",
				Value:  1,
			},
			cli.StringFlag{
				Name:   "namespace",
				EnvVar: "K3OS_OPERATOR_NAMESPACE",
				Value:  "k3os-system",
			},
		},
		Before: func(c *cli.Context) error {
			// required parameters
			if ns := c.String("namespace"); len(ns) == 0 {
				return errors.New("namespace is required")
			}
			// required uid
			if os.Getuid() != 0 {
				return fmt.Errorf("must be run as root")
			}
			// required filesystem
			if inf, err := os.Stat(system.RootDir); err != nil {
				return err
			} else if !inf.IsDir() {
				return fmt.Errorf("stat %s: not a directory", system.RootDir)
			}
			return nil
		},
		Action: Run,
	}
}

// Run the `agent` sub-command
func Run(c *cli.Context) {
	logrus.Debug("K3OS::OPERATOR >>> SETUP")

	ctx := signals.SetupSignalHandler(context.Background())

	threads := c.Int("threads")
	namespace := c.String("namespace")

	ver, err := system.GetVersion()
	if err != nil {
		logrus.Fatal(err)
	}
	if ver.Runtime != ver.Current {
		logrus.Warnf("current(%q) != runtime(%q)", ver.Current, ver.Runtime)
	}
	logrus.Infof("k3os version: previous=%s, current=%s, runtime=%s", ver.Previous, ver.Current, ver.Runtime)

	kerndir := filepath.Join(system.RootDir, "kernel")
	if kdirinf, err := os.Stat(kerndir); err != nil {
		logrus.Warn(err)
	} else if !kdirinf.IsDir() {
		logrus.Warnf("%s is not a directory", kerndir)
	}
	kernver, err := system.GetKernelVersion()
	if err != nil {
		logrus.Warn(err)
	}
	logrus.Infof("kernel version: previous=%s, current=%s, runtime=%s", kernver.Previous, kernver.Current, kernver.Runtime)

	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Fatalf("Error initializing in-cluster config: %s", err.Error())
	}

	k3osFactory, err := k3os.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		logrus.Fatalf("Error building k3OS controllers: %s", err.Error())
	}

	coreFactory, err := core.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		logrus.Fatalf("Error building core controllers: %s", err.Error())
	}

	batchFactory, err := batch.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		logrus.Fatalf("Error building rbac controllers: %s", err.Error())
	}

	logrus.Debug("K3OS::OPERATOR >>> REGISTER")

	updateChannelController := k3osFactory.K3os().V1().UpdateChannel()
	controller.Register(ctx,
		updateChannelController,
		coreFactory.Core().V1().Node(),
		batchFactory.Batch().V1().Job(),
	)

	if err := start.All(ctx, threads, k3osFactory, coreFactory, batchFactory); err != nil {
		logrus.Fatalf("Error starting: %s", err.Error())
	}

	if list, err := updateChannelController.List(namespace, metav1.ListOptions{Limit: 1}); err != nil {
		logrus.Warnf("Failed to create default UpdateChannel: %v", err)
	} else if len(list.Items) == 0 {
		if upchan, err := updateChannelController.Create(&v1.UpdateChannel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "github-releases",
				Namespace: namespace,
				Annotations: map[string]string{
					"k3os.io/node": os.Getenv("K3OS_OPERATOR_NODE"),
				},
			},
			Spec: v1.UpdateChannelSpec{
				URL:         "github-releases://rancher/k3os",
				Concurrency: 1,
				Version:     ver.Runtime,
			},
		}); err != nil {
			logrus.Warn(err)
		} else {
			logrus.Infof("Created default UpdateChannel: name=%s, url=%s, version=%s, concurrency=%d",
				upchan.ObjectMeta.Name,
				upchan.Spec.URL,
				upchan.Spec.Version,
				upchan.Spec.Concurrency,
			)
		}
	}
	<-ctx.Done()
}
