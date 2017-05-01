package model

type Image struct {
	AccountID          int                `json:"accountId"`
	Checksum           interface{}        `json:"checksum"`
	Created            int64              `json:"created"`
	Data               ImageData          `json:"data"`
	Description        interface{}        `json:"description"`
	ID                 int                `json:"id"`
	IsPublic           bool               `json:"isPublic"`
	Name               string             `json:"name"`
	PhysicalSizeBytes  interface{}        `json:"physicalSizeBytes"`
	Prepopulate        bool               `json:"prepopulate"`
	PrepopulateStamp   string             `json:"prepopulateStamp"`
	RemoveTime         interface{}        `json:"removeTime"`
	Removed            interface{}        `json:"removed"`
	State              string             `json:"state"`
	Type               string             `json:"type"`
	URL                interface{}        `json:"url"`
	UUID               string             `json:"uuid"`
	VirtualSizeBytes   interface{}        `json:"virtualSizeBytes"`
	RegistryCredential RegistryCredential `json:"registryCredential"`
	ProcessData        ProcessData
}

type ImageData struct {
	Fields      ImageFields
	DockerImage DockerImage
}

type ImageFields struct {
	Build BuildOptions
}

type BuildOptions struct {
	Context string
	FileObj string
	Remote  string
	Tag     string
}

type RegistryCredential struct {
	PublicValue string
	SecretValue string
	Data        CredentialData
}

type CredentialData struct {
	Fields CredentialFields
}

type CredentialFields struct {
	ServerAddress string
	Email         string
}

type DockerImage struct {
	FullName string `json:"fullName"`
	Server   string `json:"server"`
}

type RepoTag struct {
	Repo string
	Tag  string
	UUID string
}
