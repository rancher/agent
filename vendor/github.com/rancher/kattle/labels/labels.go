package labels

import (
	"fmt"
	"strings"
)

const (
	KattleLabel   = "io.rancher.kattle"
	RevisionLabel = "io.rancher.revision"

	GlobalLabel               = "io.rancher.scheduler.global"
	HostAffinityLabel         = "io.rancher.scheduler.affinity:host_label"
	HostAntiAffinityLabel     = "io.rancher.scheduler.affinity:host_label_ne"
	HostSoftAffinityLabel     = "io.rancher.scheduler.affinity:host_label_soft"
	HostSoftAntiAffinityLabel = "io.rancher.scheduler.affinity:host_label_soft_ne"
)

func Parse(label interface{}) map[string]string {
	labelMap := map[string]string{}
	kvPairs := strings.Split(fmt.Sprint(label), ",")
	for _, kvPair := range kvPairs {
		kv := strings.SplitN(kvPair, "=", 2)
		if len(kv) > 1 {
			labelMap[kv[0]] = kv[1]
		}
	}
	return labelMap
}
