package model

type VolumeStoragePoolMap struct {
	ID                  int         `json:"id"`
	Removed             interface{} `json:"removed"`
	State               string      `json:"state"`
	StoragePool         StoragePool
	StoragePoolID       int         `json:"storagePoolId"`
	StoragePoolLocation interface{} `json:"storagePoolLocation"`
	Volume              Volume
	VolumeID            int `json:"volumeId"`
}
