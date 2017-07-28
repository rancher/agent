//+build !windows

package hostInfo

import (
	"bufio"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

func getKernelVersion() (string, error) {
	file, err := os.Open("/proc/version")
	defer file.Close()
	data := []string{}
	if err != nil {
		return "", errors.Wrap(err, "failed to open process version file")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	version := regexp.MustCompile("\\d+.\\d+.\\d+").FindString(data[0])
	return version, nil
}
