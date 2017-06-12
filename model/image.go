package model

type Image struct {
	Data               ImageData          `json:"data"`
	RegistryCredential RegistryCredential `json:"registryCredential"`
}

type ImageData struct {
	DockerImage DockerImage
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
