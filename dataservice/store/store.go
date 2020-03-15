package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Client info
type Client struct {
	ID        string    `json:"id"`
	Stopped   bool      `json:"stopped"`
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
func ValidateClientID(db *sql.DB, userID, clientID string) (valid bool, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var client Client
	err = db.QueryRowContext(ctx, `SELECT id, stopped, userID, payload, createdAt FROM public.client WHERE id = $1`, clientID).Scan(&client.ID, &client.Stopped, &client.UserID, &client.Payload, &client.CreatedAt)
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
func SaveClient(db *sql.DB, userID, clientID, clientJSON string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := db.ExecContext(ctx, `INSERT INTO public.client (id, stopped, userID, payload) VALUES ($1, $2, $3, $4)`, clientID, false, userID, clientJSON)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("save client -- should insert one row, but inserted %d rows", rows)
	}
	return
}

// RenoClient renew client
func RenoClient(db *sql.DB, clientID, clientJSON string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := db.ExecContext(ctx, `UPDATE public.client set stopped = FALSE, payload = $2 WHERE id = $1`, clientID, clientJSON)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("renew client -- should insert one row, but inserted %d rows", rows)
	}
	return
}

// StopClient save client
func StopClient(db *sql.DB, clientID string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := db.ExecContext(ctx, `UPDATE public.client SET stopped = TRUE WHERE id = $1;`, clientID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("stop client -- should insert one row, but inserted %d rows", rows)
	}
	return
}

// PersistentMessage persistent message
func PersistentMessage(db *sql.DB, msg *Message) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messageJSON, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `INSERT INTO public.message (clientID, topic, payload) values ($1, $2, $3);`, msg.ClientID, msg.Topic, string(messageJSON))
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("persistent message -- should insert one row, but inserted %d rows", rows)
	}
	return
}
