package node

import (
	"os"
	"strings"

	"github.com/rancher/log"
	"github.com/rancher/norman/types/slice"
)

func TokenAndURL() (string, string, error) {
	return os.Getenv("CATTLE_TOKEN"), os.Getenv("CATTLE_SERVER"), nil
}

func Params() map[string]interface{} {
	roles := split(os.Getenv("CATTLE_ROLE"))
	params := map[string]interface{}{
		"customConfig": map[string]interface{}{
			"address":         os.Getenv("CATTLE_ADDRESS"),
			"internalAddress": os.Getenv("CATTLE_INTERNAL_ADDRESS"),
			"roles":           split(os.Getenv("CATTLE_ROLE")),
		},
		"etcd":              slice.ContainsString(roles, "etcd"),
		"controlPlane":      slice.ContainsString(roles, "controlplane"),
		"worker":            slice.ContainsString(roles, "worker"),
		"requestedHostname": os.Getenv("CATTLE_NODE_NAME"),
	}

	for k, v := range params {
		if m, ok := v.(map[string]string); ok {
			for k, v := range m {
				log.Infof("Option %s=%s", k, v)
			}
		} else {
			log.Infof("Option %s=%v", k, v)
		}
	}

	return map[string]interface{}{
		"node": params,
	}
}

func split(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 1 && result[0] == "" {
		return nil
	}
	return result
}
