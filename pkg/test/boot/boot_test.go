package boot

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/rancher/k3os/pkg/test"
	"github.com/rancher/k3os/pkg/test/qemu"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	itest bool
)

func TestMain(m *testing.M) {
	flag.BoolVar(&itest, "integration", false, "")
	flag.Parse()
	status := m.Run()
	os.Exit(status)
}

func TestBoot(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	if !itest {
		t.Skipf("-integration=false")
	}

	kernel, initrd, iso := test.Artifacts(t)

	spec.Run(t, "ISO", func(t *testing.T, when spec.G, it spec.S) {
		it("Default", func() {
			t.SkipNow() // need to scan console=tty1 output for something consistent. until then, skip it
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := qemu.Run(ctx,
				qemu.WithVerboseFlag("test.v"),
				qemu.WithDrive(iso),
			)
			if err != nil {
				t.Error(err)
			}
		})
	})

	spec.Run(t, "Kernel+InitRD", func(t *testing.T, when spec.G, it spec.S) {
		when("k3os.mode=live", func() {
			for run, param := range map[string]string{
				"init-cmd-poweroff": `init_cmd="poweroff -f"`,
				"boot-cmd-poweroff": `boot_cmd="poweroff -f"`,
				"run-cmd-poweroff":  `run_cmd="poweroff -f"`,
			} {
				it(run, func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
					defer cancel()
					cmd, err := qemu.Cmd(ctx,
						qemu.WithVerboseFlag("test.v"),
						qemu.WithLinux(kernel, initrd, test.Params(`k3os.mode=live`, param)...),
						qemu.WithDrive(iso),
					)
					if err != nil {
						t.Error(err)
					}
					if v := flag.Lookup("test.v"); v != nil {
						if verbose, ok := v.Value.(flag.Getter).Get().(bool); ok && verbose {
							t.Logf("EXEC%q", cmd.Args)
						}
					}
					err = cmd.Run()
					if err != nil {
						t.Error(err)
					}
				})
			}
		})
	}, spec.Report(report.Terminal{}))
}
