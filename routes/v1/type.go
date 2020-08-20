package v1

type jsonError struct {
	Error       string `json:"error"`
	ErrorObject error  `json:"extra_info"`
}
