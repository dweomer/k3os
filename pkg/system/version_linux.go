// +build linux

package system

import (
	"bytes"

	"golang.org/x/sys/unix"
)

func kernelVersion() (string, error) {
	utsname := unix.Utsname{}
	if err := unix.Uname(&utsname); err != nil {
		return "", err
	}
	n := bytes.IndexByte(utsname.Release[:], 0)
	return string(utsname.Release[:n]), nil
}
