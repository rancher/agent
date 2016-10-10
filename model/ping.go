package model

type PingResponse struct {
	Resources []PingResource `json:"resources,omitempty" yaml:"resources,omitempty"`
	Options   PingOptions    `json:"options,omitempty" yaml:"options,omitempty"`
}

type PingOptions struct {
	Instances bool `json:"instances,omitempty" yaml:"instances,omitempty"`
}

type PingResource struct {
	Type             string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Kind             string                 `json:"kind,omitempty" yaml:"kind,omitempty"`
	HostName         string                 `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	CreateLabels     map[string][]string    `json:"createLabels,omitempty" yaml:"createLabels,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	UUID             string                 `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	PhysicalHostUUID string                 `json:"physicalHostUuid,omitempty" yaml:"physicalHostUuid,omitempty"`
	Info             map[string]interface{} `json:"info,omitempty" yaml:"info,omitempty"`
	HostUUID         string                 `json:"hostUuid,omitempty" yaml:"hostUuid,omitempty"`
	Name             string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Addresss         string                 `json:"addresss,omitempty" yaml:"addresss,omitempty"`
	APIProxy         string                 `json:"apiProxy,omitempty" yaml:"apiProxy,omitempty"`
	State            string                 `json:"state,omitempty" yaml:"state,omitempty"`
	SystemContainer  string                 `json:"systemContainer" yaml:"systemContainer"`
	DockerID         string                 `json:"dockerId,omitempty" yaml:"dockerId,omitempty"`
	Image            string                 `json:"image,omitempty" yaml:"image,omitempty"`
	Created          int64                  `json:"created,omitempty" yaml:"created,omitempty"`
	Memory           uint64                 `json:"memory,omitempty" yaml:"memory,omitempty"`
	MilliCPU         uint64                 `json:"milliCpu,omitempty" yaml:"milli_cpu,omitempty"`
	LocalStorageMb   uint64                 `json:"localStorageMb,omitempty" yaml:"local_storage_mb,omitempty"`
}
