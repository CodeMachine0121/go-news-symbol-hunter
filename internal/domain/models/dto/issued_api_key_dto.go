package dto

type IssuedApiKeyDto struct {
	Name   string `json:"name"`
	ApiKey string `json:"apiKey"`
	Status string `json:"status"`
}
