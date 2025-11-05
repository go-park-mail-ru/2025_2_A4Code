package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Body    interface{} `json:"body,omitempty"`
}

func SendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Status:  statusCode,
		Message: errorMsg,
		Body:    struct{}{},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
