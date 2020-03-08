package store

import (
	"context"
	"database/sql"
	"time"
)

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
func ValidateClientID(db *sql.DB, userID, clientID string) (bool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var client Client
	err := db.QueryRowContext(ctx, `SELECT userID FROM public.client WHERE id = $1;`, clientID).Scan(&client)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if client.UserID == userID {
		return true, nil
	}
	return false, nil
}

// SaveClient save client
func SaveClient(db *sql.DB, userID string) (err error) {
	return
}

// PersistentMessage persistent message
func PersistentMessage(db *sql.DB, msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `insert into t_message (message) values ($1);`, msg)
	return err
}
