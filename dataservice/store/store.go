package store

import "time"

// Client info
type Client struct {
	ID        string    `json:"id"`
	Closed    bool      `json:"closed"`
	UserID    string    `json:"userID"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

// Message info
type Message struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"clientID"`
	Topic     string    `json:"topic"`
	Payload   []byte    `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

// ValidateClientID validate clientID
func ValidateClientID(userID, clientID string) bool {
	return true
}

// SaveClient save client
func SaveClient(userID string) (err error) {
	return
}
