package model

type Volume struct {
	AccountID int `json:"accountId"`
	AllocationState string `json:"allocationState"`
	AttachedState string `json:"attachedState"`
	Created int64 `json:"created"`
	Data map[string]interface{}
	Description interface{} `json:"description"`
	DeviceNumber int `json:"deviceNumber"`
	Format interface{} `json:"format"`
	ID int `json:"id"`
	ImageID int `json:"imageId"`
	InstanceID int `json:"instanceId"`
	Name interface{} `json:"name"`
	Offering interface{} `json:"offering"`
	OfferingID interface{} `json:"offeringId"`
	PhysicalSizeBytes interface{} `json:"physicalSizeBytes"`
	Recreatable bool `json:"recreatable"`
	RemoveTime interface{} `json:"removeTime"`
	Removed interface{} `json:"removed"`
	State string `json:"state"`
	Type string `json:"type"`
	UUID string `json:"uuid"`
	VirtualSizeBytes interface{} `json:"virtualSizeBytes"`
}
