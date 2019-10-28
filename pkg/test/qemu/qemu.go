package qemu

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Run for QEMU wraps exec.CommandContext().Run()
func Run(ctx context.Context, opts ...Opt) error {
	cmd, err := Cmd(ctx, opts...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// Cmd for QEMU wraps exec.CommandContext
func Cmd(ctx context.Context, opts ...Opt) (*exec.Cmd, error) {
	var opt Options
	var def = []Opt{
		DefaultArch(),
		DefaultMemory(),
		DefaultSMP(),
		DefaultRTC(),
		DefaultMachine(),
		DefaultDisplay(),
		DefaultSerial(),
		DefaultMonitor(),
	}
	for _, app := range append(def, opts...) {
		if err := app(&opt); err != nil {
			return nil, err
		}
	}

	cmd := exec.CommandContext(ctx, `qemu-system-`+opt.Arch,
		"-rtc", `base=`+opt.RTC.Base+`,clock=`+opt.RTC.Clock,
		"-serial", opt.Serial,
		"-display", opt.Display,
		"-monitor", opt.Monitor,
	)

	if opt.CPU != "" {
		cmd.Args = append(cmd.Args, "-cpu", opt.CPU)
	}

	if opt.Memory > 0 {
		cmd.Args = append(cmd.Args, "-m", fmt.Sprintf("%d", opt.Memory))
	}

	var smp string
	if opt.SMP.CPUs > 0 {
		smp += fmt.Sprintf("cpus=%d", opt.SMP.CPUs)
	}
	if opt.SMP.Cores > 0 {
		smp += fmt.Sprintf(",cores=%d", opt.SMP.Cores)
	}
	if opt.SMP.Threads > 0 {
		smp += fmt.Sprintf(",threads=%d", opt.SMP.Threads)
	}
	smp = strings.TrimLeft(smp, `,`)
	if len(smp) > 0 {
		cmd.Args = append(cmd.Args, "-smp", smp)
	}

	if opt.Machine.Name != "" || len(opt.Machine.Accel) > 0 {
		machine := opt.Machine.Name
		for i, accel := range opt.Machine.Accel {
			if i == 0 {
				machine += `,accel=`
			}
			machine += accel + `:`
		}
		machine = strings.TrimLeft(machine, `,`)
		machine = strings.TrimRight(machine, `:`)
		cmd.Args = append(cmd.Args, "-machine", machine)
	}

	for _, d := range opt.Drives {
		cmd.Args = append(cmd.Args, "-drive", d)
	}

	if opt.Linux.Kernel != "" {
		cmd.Args = append(cmd.Args, "-kernel", opt.Linux.Kernel)
	}
	if opt.Linux.InitRD != "" {
		cmd.Args = append(cmd.Args, "-initrd", opt.Linux.InitRD)
	}
	if len(opt.Linux.CmdLine) > 0 {
		cmd.Args = append(cmd.Args, "-append", strings.Join(opt.Linux.CmdLine, ` `))
	}

	cmd.Dir = opt.Command.Dir
	cmd.Env = opt.Command.Env
	cmd.Stdin = opt.Command.Stdin
	cmd.Stdout = opt.Command.Stdout
	cmd.Stderr = opt.Command.Stderr

	return cmd, nil
}
