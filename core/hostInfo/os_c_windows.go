//+build windows

package hostInfo

import (
	"bufio"
	"fmt"
	"os/exec"
)

func getKernelVersion() (string, error) {
	var major, minor string
	var err error
	major, err = getKernelVersionPart("Major")
	if err != nil {
		return "", err
	}
	minor, err = getKernelVersionPart("Minor")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", major, minor), nil
}

func getKernelVersionPart(part string) (string, error) {
	cmd := exec.Command("powershell.exe", fmt.Sprintf("[System.Environment]::OSVersion.Version.%s", part))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	var value []byte
	reader := bufio.NewReader(stdout)
	value, _, err = reader.ReadLine()
	if err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return string(value), nil
}

func getOSName() string {
	return "windows"
}
