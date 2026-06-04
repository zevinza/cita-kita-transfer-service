package model

// ErrorResponse is returned when a request fails.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}

type Response struct {
	Code    int    `json:"code"`
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
