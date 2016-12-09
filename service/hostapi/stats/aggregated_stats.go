package stats

import (
	"encoding/json"
	"io"
)

type AggregatedStats []AggregatedStat

type AggregatedStat struct {
	ID           string `json:"id,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	MemLimit     uint64 `json:"memLimit,omitempty"`
	containerStats
}

func convertToAggregatedStats(id string, containerIds map[string]string, resourceType string, stats []containerInfo, memLimit uint64) []AggregatedStats {
	totalAggregatedStats := []AggregatedStats{}
	if len(stats) == 0 {
		return totalAggregatedStats
	}

	totalAggregatedStat := []AggregatedStat{}
	for j := 0; j < len(stats); j++ {
		aggStats := AggregatedStat{id, resourceType, memLimit, stats[j].Stats[0]}
		if id == "" {
			aggStats.ID = containerIds[stats[j].ID]
		}
		totalAggregatedStat = append(totalAggregatedStat, aggStats)
	}
	totalAggregatedStats = append(totalAggregatedStats, totalAggregatedStat)

	return totalAggregatedStats
}

func writeAggregatedStats(id string, containerIds map[string]string, resourceType string, infos []containerInfo, memLimit uint64, writer io.Writer) error {
	aggregatedStats := convertToAggregatedStats(id, containerIds, resourceType, infos, memLimit)
	for _, stat := range aggregatedStats {
		data, err := json.Marshal(stat)
		if err != nil {
			return err
		}

		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}

	return nil
}
