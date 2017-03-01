package model

//ProxyError structure contains the error resource definition
type ProxyError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
