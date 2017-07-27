package model

type Volume struct {
	AccountID       int    `json:"accountId"`
	AllocationState string `json:"allocationState"`
	AttachedState   string `json:"attachedState"`
	Created         int64  `json:"created"`
	Data            VolumeData
	Description     interface{} `json:"description"`
	Format          string      `json:"format"`
	ID              int         `json:"id"`
	Image           Image       `json:"image"`
	ImageID         interface{} `json:"imageId"`
	Instance        Instance    `json:"instance"`
	InstanceID      interface{} `json:"instanceId"`
	Kind            string      `json:"kind"`
	Name            string      `json:"name"`
	Offering        interface{} `json:"offering"`
	OfferingID      interface{} `json:"offeringId"`
	PhysicalSizeMb  interface{} `json:"physicalSizeMb"`
	RemoveTime      int64       `json:"removeTime"`
	Removed         int64       `json:"removed"`
	State           string      `json:"state"`
	Type            string      `json:"type"`
	URI             string      `json:"uri"`
	UUID            string      `json:"uuid"`
	VirtualSizeMb   interface{} `json:"virtualSizeMb"`
	ZoneID          int         `json:"zoneId"`
	Driver          string
}

type VolumeData struct {
	Fields VolumeFields
}

type VolumeFields struct {
	Driver     string
	DriverOpts map[string]string
	IsHostPath bool
}
