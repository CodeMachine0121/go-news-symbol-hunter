package dto

type IssuedApiKeyDto struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	ApiKey string `json:"apiKey"`
	Status string `json:"status"`
}
