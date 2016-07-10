package model

type Instance struct {
	AccountID int `json:"accountId"`
	AllocationState string `json:"allocationState"`
	Created int64 `json:"created"`
	Data map[string]interface{}
	Ports []Port
	Description interface{} `json:"description"`
	Hostname interface{} `json:"hostname"`
	ID int `json:"id"`
	Image Image
	ImageID int `json:"imageId"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Nics []Nic
	Offering interface{} `json:"offering"`
	OfferingID interface{} `json:"offeringId"`
	OnCrash string `json:"onCrash"`
	PostComputeState string `json:"postComputeState"`
	RemoveTime interface{} `json:"removeTime"`
	Removed interface{} `json:"removed"`
	RequestedOfferingID interface{} `json:"requestedOfferingId"`
	RequestedState interface{} `json:"requestedState"`
	State string `json:"state"`
	Type string `json:"type"`
	UUID string `json:"uuid"`
	Volumes []Volume
	ZoneID int `json:"zoneId"`
	ExternalId string
	AgentId int
	InstanceLinks []Link
	NetworkContainer Instance
	NativeContainer bool
	SystemContainer string
	DataVolumesFromContainers []Instance
	Command_args []string
	Labels map[string]interface{}
}
