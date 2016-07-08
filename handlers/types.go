package handlers

import (
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
)

type Event struct {
	Data map[string]interface{}
	ID string `json:"id"`
	Name string `json:"name"`
	PreviousIds interface{} `json:"previousIds"`
	PreviousNames interface{} `json:"previousNames"`
	Publisher interface{} `json:"publisher"`
	ReplyTo string `json:"replyTo"`
	ResourceID string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Time int64 `json:"time"`
}

type InstanceHostMap struct {
	Host Host
	HostID int `json:"hostId"`
	ID int `json:"id"`
	Instance Instance
	InstanceID int `json:"instanceId"`
	Removed interface{} `json:"removed"`
	State string `json:"state"`
	Type string `json:"type"`
}

type Host struct {
	AgentID int `json:"agentId"`
	Created int64 `json:"created"`
	Data map[string]interface{}
	Description interface{} `json:"description"`
	ID int `json:"id"`
	Kind string `json:"kind"`
	Name interface{} `json:"name"`
	Removed interface{} `json:"removed"`
	State string `json:"state"`
	Type string `json:"type"`
	UUID string `json:"uuid"`
}

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
	systemContainer string
	DataVolumesFromContainers []Instance
	Command_args []string
	Labels map[string]interface{}
}

type Container types.Container

type Options types.ContainerListOptions

type Tuple struct {
	Src, Dest string
}

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

type Port struct {
	Protocol string `json:"protocol"`
	PrivatePort int `json:"privatePort"`
	PublicPort int `json:"publicPort"`
}

type Image struct {
	AccountID int `json:"accountId"`
	Checksum interface{} `json:"checksum"`
	Created int64 `json:"created"`
	Data map[string]interface{}
	Description interface{} `json:"description"`
	ID int `json:"id"`
	IsPublic bool `json:"isPublic"`
	Name string `json:"name"`
	PhysicalSizeBytes interface{} `json:"physicalSizeBytes"`
	Prepopulate bool `json:"prepopulate"`
	PrepopulateStamp string `json:"prepopulateStamp"`
	RemoveTime interface{} `json:"removeTime"`
	Removed interface{} `json:"removed"`
	State string `json:"state"`
	Type string `json:"type"`
	URL interface{} `json:"url"`
	UUID string `json:"uuid"`
	VirtualSizeBytes interface{} `json:"virtualSizeBytes"`
	RegistryCredential map[string]interface{}
}

type DockerImage struct{
	FullName string `json:"fullName"`
	ID string `json:"id"`
	Namespace string `json:"namespace"`
	QualifiedName string `json:"qualifiedName"`
	Repository string `json:"repository"`
	Tag string `json:"tag"`
}

type Nic struct {
	MacAddress string `json:"macAddress"`
	DeviceNumber int `json:"deviceNumber"`
	IPAddresses []struct {
		Address string `json:"address"`
		Role string `json:"role"`
		Subnet struct {
				CidrSize int `json:"cidrSize"`
				NetworkAddress string
			} `json:"subnet"`
	} `json:"ipAddresses"`
	Network struct {
			   Kind string `json:"kind"`
			   NetworkServices []Service
		   } `json:"network"`
}

type Host_Bind struct {
	bind_addr string
	publicPort int
}

type Service struct {
	Kind string
}

type Link struct {
	TargetInstanceId string
	LinkName string
	TargetInstance Instance
	Data map[string]interface{}
}

type VolumeCreateRequest types.VolumeCreateRequest

type Host_Config container.HostConfig

type Option_Config struct {
	Key string
	Dev_List []map[string]string
	Docker_Field string
	Field string
}

type Image_Params struct {
	Image Image
	Tag string
	Mode string
	Complete bool
}

type Progress struct {

}

type Config container.Config

type AuthConfig types.AuthConfig
