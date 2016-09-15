package model

type Tuple struct {
	Src, Dest string
}

type OptionConfig struct {
	Key         string
	DevList     []map[string]string
	DockerField string
	Field       string
}

type ImageParams struct {
	Image     Image
	Tag       string
	Mode      string
	Complete  bool
	ImageUUID string
}

type Method func(map[string]string, []string) (interface{}, error)
