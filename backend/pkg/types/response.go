package types

type APIErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
}

type GetInstallURLResponse struct {
	URL string `json:"url"`
}