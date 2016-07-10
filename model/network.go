package model

type Port struct {
	Protocol string `json:"protocol"`
	PrivatePort int `json:"privatePort"`
	PublicPort int `json:"publicPort"`
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
	Bind_addr string
	PublicPort int
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
