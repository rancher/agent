package model

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
