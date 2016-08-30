package model

type ImageStoragePoolMap struct {
	Created       int64 `json:"created"`
	ID            int   `json:"id"`
	Image         Image
	ImageID       int         `json:"imageId"`
	RemoveLocked  bool        `json:"removeLocked"`
	RemoveTime    interface{} `json:"removeTime"`
	Removed       interface{} `json:"removed"`
	State         string      `json:"state"`
	StoragePool   StoragePool
	StoragePoolID int         `json:"storagePoolId"`
	Type          string      `json:"type"`
	URI           interface{} `json:"uri"`
}
