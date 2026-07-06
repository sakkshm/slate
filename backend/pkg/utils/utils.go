package utils

import (
	"encoding/json"
	"net/http"
	"slate-backend/pkg/types"

	"github.com/google/uuid"
)

func GenerateRandomString(length int) (string, error){
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return uuid.String()[:length], nil
}

func WriteHTTPError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(types.APIErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}