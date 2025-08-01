package common

import (
	"log"
	"time"
)

type Response struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func createResponse(status, message string) Response {
	return Response{
		Status:    status,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func Success(message string) Response {
	return createResponse("success", message)
}

func Error(message string) Response {
	log.Printf("Error: %s", message)
	return createResponse("error", message)
}