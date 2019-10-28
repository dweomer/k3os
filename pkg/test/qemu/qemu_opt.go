package qemu

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// Options represents command-line flags passed to the qemu-system-<arch> executable.
type Options struct {
	Arch string
	CPU  string
	SMP  struct {
		CPUs    int
		Cores   int
		Threads int
	}
	RTC struct {
		Base  string
		Clock string
	}
	Memory  int
	Machine struct {
		Name  string
		Accel []string
	}
	Linux struct {
		InitRD  string
		Kernel  string
		CmdLine []string
	}
	Display string
	Serial  string
	Monitor string
	Drives  []string
	Devices []string
	Command struct {
		Dir    string
		Env    []string
		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer
	}
}

// Opt represents options for the execution of `qemu-system-<arch>`.
type Opt func(*Options) error

// WithArch determines which `qemu-system-<arch>` executable to invoke.
func WithArch(arch string) Opt {
	return func(o *Options) error {
		switch arch {
		case `amd64`:
			o.Arch = `x86_64`
		case `arm64`:
			o.Arch = `aarch64`
		default:
			o.Arch = arch
		}
		return nil
	}
}

// DefaultArch selects the `qemu-system-<arch>` matching the host architecture.
func DefaultArch() Opt {
	return WithArch(runtime.GOARCH)
}

// WithCPU is -cpu
// See `qemu-system-<arch> -cpu help`
func WithCPU(cpu string) Opt {
	return func(o *Options) error {
		o.CPU = cpu
		return nil
	}
}

// WithMemory is -m
func WithMemory(m int) Opt {
	return func(o *Options) error {
		o.Memory = m
		return nil
	}
}

// DefaultMemory is -m 2048
func DefaultMemory() Opt {
	return WithMemory(2048)
}

// WithMachine is -machine
// See `qemu-system-<arch> -machine help`
func WithMachine(name string, accel ...string) Opt {
	return func(o *Options) error {
		o.Machine.Name = name
		o.Machine.Accel = accel
		return nil
	}
}

// DefaultMachine selects the default, fully emulated machine based on target architecture.
func DefaultMachine() Opt {
	return func(o *Options) error {
		accel := []string{`kvm`, `hvf`, `hax`, `tcg`}
		switch {
		case o.Arch == "x86_64":
			return WithMachine(`q35`, accel...)(o)
		case o.Arch == "aarch64":
			return WithMachine(`virt`, accel...)(o)
		case o.Arch == "arm":
			return WithMachine(`raspi2`, accel...)(o)
		}
		return nil
	}
}

// WithSMP is -smp
func WithSMP(cpus, cores, threads int) Opt {
	return func(o *Options) error {
		o.SMP.CPUs = cpus
		o.SMP.Cores = cores
		o.SMP.Threads = threads
		return nil
	}
}

// DefaultSMP is -smp cpus=1
func DefaultSMP() Opt {
	return WithSMP(1, 0, 0)
}

// WithRTC is -rtc
func WithRTC(base, clock string) Opt {
	return func(o *Options) error {
		o.RTC.Base = base
		o.RTC.Clock = clock
		return nil
	}
}

// DefaultRTC is -rtc base=utc,clock=rt
func DefaultRTC() Opt {
	return WithRTC("utc", "rt")
}

// WithLinux combines -kernel, -initrd, and -append
func WithLinux(kernel, initrd string, cmdline ...string) Opt {
	return func(o *Options) error {
		o.Linux.Kernel = kernel
		o.Linux.InitRD = initrd
		o.Linux.CmdLine = cmdline
		return nil
	}
}

// WithDisplay is -display
func WithDisplay(spec string) Opt {
	return func(o *Options) error {
		o.Display = spec
		return nil
	}
}

// DefaultDisplay is -display none
func DefaultDisplay() Opt {
	return WithDisplay("none")
}

// WithSerial is -serial
func WithSerial(spec string) Opt {
	return func(o *Options) error {
		o.Serial = spec
		return nil
	}
}

// DefaultSerial is -serial stdio
func DefaultSerial() Opt {
	return WithSerial("stdio")
}

// WithMonitor is -monitor
func WithMonitor(spec string) Opt {
	return func(o *Options) error {
		o.Monitor = spec
		return nil
	}
}

// DefaultMonitor is -monitor none
func DefaultMonitor() Opt {
	return WithMonitor("none")
}

// WithDrive is -drive
func WithDrive(m map[string]string) Opt {
	return func(o *Options) error {
		if len(m) > 0 {
			var d string
			for k, v := range m {
				d += k + `=` + v + `,`
			}
			o.Drives = append(o.Drives, fmt.Sprintf("%sindex=%d", d, len(o.Drives)))
		}
		return nil
	}
}

// WithDevice is -device
func WithDevice(m map[string]string) Opt {
	return func(o *Options) error {
		if len(m) > 0 {
			var d string
			for k, v := range m {
				d += k + `=` + v + `,`
			}
			o.Drives = append(o.Drives, strings.TrimRight(d, `,`))
		}
		return nil
	}
}

// WithCommandDir sets command working directory (exec.Cmd.Dir)
func WithCommandDir(dir string) Opt {
	return func(o *Options) error {
		o.Command.Dir = dir
		return nil
	}
}

// WithCommandEnv sets command environment variables (exec.Cmd.Env)
func WithCommandEnv(env ...string) Opt {
	return func(o *Options) error {
		o.Command.Env = env
		return nil
	}
}

// WithCommandIO sets command input/output
func WithCommandIO(stdin io.Reader, stdout, stderr io.Writer) Opt {
	return func(o *Options) error {
		o.Command.Stdin = stdin
		o.Command.Stdout = stdout
		o.Command.Stderr = stderr
		return nil
	}
}

// WithVerboseFlag if flag is set, attach to stdout/stderr
func WithVerboseFlag(f string) Opt {
	return func(o *Options) error {
		if v := flag.Lookup(f); v != nil {
			if verbose, ok := v.Value.(flag.Getter).Get().(bool); ok && verbose {
				o.Command.Stdout = os.Stdout
				o.Command.Stderr = os.Stderr
			}
		}
		return nil
	}
}
