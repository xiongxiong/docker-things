package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_loadClients(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockClient := func(index int) client {
		return client{
			Username: fmt.Sprintf("username_%d", index),
			Password: fmt.Sprintf("password_%d", index),
			Brokers:  []string{"localhost:1883"},
			Topics: []topic{
				topic{
					Topic: "topicA",
					Qos:   byte(2),
				},
				topic{
					Topic: "topicB",
					Qos:   byte(2),
				},
			},
		}
	}
	rows := sqlmock.NewRows([]string{"id", "userID", "payload"})
	rb0 := reqbodySubscribe{
		ClientID: "id_0",
		Client:   mockClient(0),
	}
	jsonStr0, err := json.Marshal(rb0.Client)
	if err != nil {
		t.Fatalf("unable to marshal client, %s", err)
	}
	rows.AddRow(rb0.ClientID, "userID0", jsonStr0)
	rb1 := reqbodySubscribe{
		ClientID: "id_1",
		Client:   mockClient(1),
	}
	jsonStr1, err := json.Marshal(rb1.Client)
	if err != nil {
		t.Fatalf("unable to marshal client, %s", err)
	}
	rows.AddRow(rb1.ClientID, "userID1", jsonStr1)
	mock.ExpectQuery("^SELECT id, userID, payload FROM public.client WHERE stopped = 'false'$").WillReturnRows(rows)

	type args struct {
		db *sql.DB
	}
	tests := []struct {
		name string
		args args
		want []reqbodySubscribe
	}{
		{
			name: "load",
			args: args{
				db: db,
			},
			want: []reqbodySubscribe{
				rb0,
				rb1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loadClients(tt.args.db); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadClients() = %v, want %v", got, tt.want)
			}
		})
	}
}
