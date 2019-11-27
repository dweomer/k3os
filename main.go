package main

// Copyright 2019 Rancher Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/mount"
	"github.com/docker/docker/pkg/reexec"
	"github.com/rancher/k3os/pkg/root"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	reexec.Register("/init", initrd)      // mode=live
	reexec.Register("/sbin/init", initrd) // mode=local
	reexec.Register("(enter-root)", root.Enter)

	if !reexec.Init() {
		app := cli.NewApp()
		args := []string{app.Name}
		path := filepath.Base(os.Args[0])
		if path != app.Name && app.Command(path) != nil {
			args = append(args, path)
		}
		args = append(args, os.Args[1:]...)
		// this will bomb if the app has any non-defaulted, required flags
		err := app.Run(args)
		if err != nil {
			logrus.Fatal(err)
		}
	}
}

func initrd() {
	if err := root.ProcFS(); err != nil {
		logrus.Fatalf("failed to mount /proc: %v", err)
	}
	root.SetDebug("k3os.debug")
	if err := root.DevFS(); err != nil {
		logrus.Fatalf("failed to mount /dev: %v", err)
	}

	root.Relocate()

	if err := mount.Mount("", "/", "none", "rw,remount"); err != nil {
		logrus.Errorf("failed to remount root as rw: %v", err)
	}

	if err := root.Mount("./k3os/data", os.Args, os.Stdout, os.Stderr); err != nil {
		logrus.Fatalf("failed to enter root: %v", err)
	}
}
