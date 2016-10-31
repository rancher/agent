package model

type Port struct {
	Protocol    string `json:"protocol"`
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort"`
	Data        PortData
	IPAddress   string
}

type PortData struct {
	Fields PortFields
}

type PortFields struct {
	BindAddress string
}

type Nic struct {
	MacAddress   string `json:"macAddress"`
	DeviceNumber int    `json:"deviceNumber"`
	IPAddresses  []struct {
		Address string `json:"address"`
		Role    string `json:"role"`
		Subnet  struct {
			CidrSize       int `json:"cidrSize"`
			NetworkAddress string
		} `json:"subnet"`
	} `json:"ipAddresses"`
	Network struct {
		Name            string `json:"name"`
		Kind            string `json:"kind"`
		NetworkServices []Service
	} `json:"network"`
}

type HostBind struct {
	BindAddr   string
	PublicPort int
}

type Service struct {
	Kind string
}

type Link struct {
	Type             string
	TargetInstanceID int
	LinkName         string
	TargetInstance   Instance
	Data             LinkData
}

type LinkData struct {
	Fields LinkFields
}

type LinkFields struct {
	InstanceNames []string
	Ports         []struct {
		Protocol    string
		PrivatePort interface{}
	}
}
