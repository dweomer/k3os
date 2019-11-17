// +build !linux

package system

func kernelVersion() (string, error) {
	panic("KERNEL_VERSION")
	return "", nil
}
