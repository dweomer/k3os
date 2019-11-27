package root

import (
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
)

// SetDebug looks for the presence of the specified kernel parameter to enable debug logging
func SetDebug(param string) {
	if param == "" {
		return
	}
	if bytes, err := ioutil.ReadFile("/proc/cmdline"); err != nil {
		logrus.Warn(err)
	} else {
		for _, word := range strings.Fields(string(bytes)) {
			if word == param {
				logrus.SetLevel(logrus.DebugLevel)
			}
		}
	}
}
