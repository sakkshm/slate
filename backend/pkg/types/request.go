package types

type CallbackRequest struct {
	Code           string `json:"code"`
	InstallationID string `json:"installation_id"`
	State          string `json:"state"`
}