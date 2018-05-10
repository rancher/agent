package haproxy

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/leodotcloud/log"
)

type Stat map[string]string

type Monitor struct {
	SocketPath string
}

func (h *Monitor) Stats() ([]Stat, error) {
	stats := []Stat{}

	lines, err := h.runCommand("show stat")
	if err != nil {
		return nil, err
	}

	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") {
		return nil, fmt.Errorf("Failed to find stats")
	}

	keys := strings.Split(strings.TrimPrefix(lines[0], "# "), ",")

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		values := strings.Split(line, ",")
		if len(keys) != len(values) {
			log.Errorf("Invalid stat line: %s", line)
		}

		stat := Stat{}

		for i := 0; i < len(values); i++ {
			stat[keys[i]] = values[i]
		}

		stats = append(stats, stat)
	}

	return stats, err
}

func (h *Monitor) runCommand(cmd string) ([]string, error) {
	conn, err := net.Dial("unix", h.SocketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd + "\n"))
	if err != nil {
		return nil, err
	}

	result := []string{}
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}

	return result, scanner.Err()
}
