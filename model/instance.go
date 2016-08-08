package model

type Instance struct {
	AccountID                   int    `json:"accountId"`
	AllocationState             string `json:"allocationState"`
	Created                     int64  `json:"created"`
	Data                        map[string]interface{}
	Ports                       []Port
	Description                 string `json:"description"`
	Hostname                    string `json:"hostname"`
	ID                          int    `json:"id"`
	Image                       Image
	ImageID                     int    `json:"imageId"`
	Kind                        string `json:"kind"`
	Name                        string `json:"name"`
	Nics                        []Nic
	Offering                    interface{} `json:"offering"`
	OfferingID                  interface{} `json:"offeringId"`
	OnCrash                     string      `json:"onCrash"`
	PostComputeState            string      `json:"postComputeState"`
	RemoveTime                  interface{} `json:"removeTime"`
	Removed                     interface{} `json:"removed"`
	RequestedOfferingID         interface{} `json:"requestedOfferingId"`
	RequestedState              interface{} `json:"requestedState"`
	State                       string      `json:"state"`
	Type                        string      `json:"type"`
	UUID                        string      `json:"uuid"`
	Volumes                     []Volume
	ZoneID                      int `json:"zoneId"`
	ExternalID                  string
	AgentID                     int
	InstanceLinks               []Link
	NetworkContainer            map[string]interface{}
	NativeContainer             bool
	SystemContainer             string
	DataVolumesFromContainers   []map[string]interface{}
	CommandArgs                 []string
	Labels                      map[string]interface{}
	ProcessData                 map[string]interface{}
	VolumesFromDataVolumeMounts []Volume
	Token                       string
}
