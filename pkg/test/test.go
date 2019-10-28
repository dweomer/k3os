package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func Params(extra ...string) []string {
	defaults := []string{
		`loglevel=5`,
		`panic=-1`,
		`console=ttyS0`,
		`k3os.net.noping`,
	}
	return append(defaults, extra...)
}

func Artifacts(t *testing.T) (kernel, initrd string, iso map[string]string) {
	kernel = os.Getenv("K3OS_TEST_KERNEL")
	initrd = os.Getenv("K3OS_TEST_INITRD")
	iso = map[string]string{
		"if":    "ide",
		"media": "cdrom",
		"file":  os.Getenv("K3OS_TEST_ISO"),
	}
	if kernel == "" || initrd == "" {
		if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
			var dir string
			dir, err = os.Getwd()
			if err == nil {
				dir, err = filepath.Rel(dir, strings.TrimSpace(string(out)))
			}
			if err != nil {
				dir = strings.TrimSpace(string(out))
			}
			dir = filepath.Join(dir, "dist", "artifacts")
			if inf, err := os.Stat(dir); err == nil && inf.IsDir() {
				kernel = filepath.Join(dir, `k3os-vmlinuz-`+runtime.GOARCH)
				initrd = filepath.Join(dir, `k3os-initrd-`+runtime.GOARCH)
				iso[`file`] = filepath.Join(dir, `k3os-`+runtime.GOARCH+`.iso`)
			}
		}
	}
	if kernel == "" || initrd == "" {
		dir := `/output`
		if inf, err := os.Stat(dir); err == nil && inf.IsDir() {
			kernel = filepath.Join(dir, `vmlinuz`)
			initrd = filepath.Join(dir, `initrd`)
			iso[`file`] = filepath.Join(dir, `k3os.iso`)
		}
	}
	if kernel == "" || initrd == "" || iso[`file`] == "" {
		t.Skip("missing kernel/initrd/iso")
	}
	return kernel, initrd, iso
}
