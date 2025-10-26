package response

import (
	"net/http"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func OK() Response {
	return Response{Status: http.StatusText(http.StatusOK)}
}

func Error(msg string) Response {
	return Response{Status: http.StatusText(http.StatusBadRequest), Message: msg}
}
