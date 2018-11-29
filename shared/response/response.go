package response

import (
	"time"
)

type Error struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type Response struct {
	Success   bool        `json:"success"`
	Errors    []*Error    `json:"errors,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp string      `json:"ts,omitempty"`
}

func SuccessResponse(payload interface{}) *Response {
	return &Response{
		Success:   true,
		Payload:   payload,
		Timestamp: time.Now().Format("2006-01-02T15:04:05"),
	}
}

func ErrorResponse(errors ...*Error) *Response {
	return &Response{
		Success:   false,
		Errors:    errors,
		Timestamp: time.Now().Format("2006-01-02T15:04:05"),
	}
}
