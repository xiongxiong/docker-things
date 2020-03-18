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
	ID       string    `json:"id"`
	Stopped  bool      `json:"stopped"`
	Payload  string    `json:"payload"`
	CreateAt time.Time `json:"create_at"`
	UpdateAt time.Time `json:"update_at"`
}

// Message info
type Message struct {
	ID       string    `json:"id"`
	ClientID string    `json:"device_id"`
	Topic    string    `json:"topic"`
	Payload  []byte    `json:"payload"`
	CreateAt time.Time `json:"create_at"`
}

// ValiClient validate client token
func ValiClient(db *sql.DB, clientID, token string) (isValid bool, err error) {
	return true, nil
}

// SaveClient save client
func SaveClient(db *sql.DB, clientID, clientJSON string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := db.ExecContext(ctx, `INSERT INTO public.connection (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET stopped = FALSE, payload = $2, update_at = $3`, clientID, clientJSON, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("save client -- should upsert one row, but upserted %d rows", rows)
	}
	return
}

// StopClient stop client
func StopClient(db *sql.DB, clientID string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := db.ExecContext(ctx, `UPDATE public.connection SET stopped = TRUE, update_at = $2 WHERE id = $1`, clientID, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("stop client -- should update one row, but updated %d rows", rows)
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
	result, err := db.ExecContext(ctx, `INSERT INTO public.message (device_id, topic, payload) values ($1, $2, $3);`, msg.ClientID, msg.Topic, string(messageJSON))
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
