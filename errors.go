package endee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Base API Error
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Endee API Error %d: %s", e.StatusCode, e.Message)
}

// Specific Error Types
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Authentication Error: %s", e.Message)
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("Resource Not Found: %s", e.Message)
}

type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("Forbidden: %s", e.Message)
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("Conflict: %s", e.Message)
}

type SubscriptionError struct {
	Message string
}

func (e *SubscriptionError) Error() string {
	return fmt.Sprintf("Subscription Error: %s", e.Message)
}

type ServerError struct {
	Message string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("Server Busy: %s", e.Message)
}

// checkError checks the response status code and returns a corresponding error if not 200 OK
func checkError(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Read body to get error message
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	// Try to parse JSON error message {"error": "msg"}
	var errorResp map[string]interface{}
	var msg string
	if jsonErr := json.Unmarshal(bodyBytes, &errorResp); jsonErr == nil {
		if val, ok := errorResp["error"].(string); ok {
			msg = val
		} else {
			msg = string(bodyBytes)
		}
	} else {
		// Fallback to raw text
		msg = string(bodyBytes)
		if msg == "" {
			msg = "Unknown error"
		}
	}

	switch resp.StatusCode {
	case 400:
		return &APIError{StatusCode: 400, Message: msg}
	case 401:
		return &AuthenticationError{Message: msg}
	case 402:
		return &SubscriptionError{Message: msg}
	case 403:
		return &ForbiddenError{Message: msg}
	case 404:
		return &NotFoundError{Message: msg}
	case 409:
		return &ConflictError{Message: msg}
	case 500, 502, 503, 504:
		return &ServerError{Message: "Server is busy. Please try again in sometime"}
	default:
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}
}
