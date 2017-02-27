package model

const SignatureHeader = "X-API-Auth-Signature"

//FilterData defines the properties of a pre/post API filter
type FilterData struct {
	Type        string   `json:"type"`
	Endpoint    string   `json:"endpoint"`
	SecretToken string   `json:"secretToken"`
	Methods     []string `json:"methods"`
	Paths       []string `json:"paths"`
	Timeout     string   `json:"timeout"`
}

//APIRequestData defines the properties of a API Request/Response Body sent to/from a filter
type APIRequestData struct {
	Headers   map[string][]string    `json:"headers,omitempty"`
	Body      map[string]interface{} `json:"body,omitempty"`
	UUID      string                 `json:"UUID,omitempty"`
	APIPath   string                 `json:"APIPath,omitempty"`
	APIMethod string                 `json:"APIMethod,omitempty"`
	EnvID     string                 `json:"envID,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Message   string                 `json:"message,omitempty"`
}
