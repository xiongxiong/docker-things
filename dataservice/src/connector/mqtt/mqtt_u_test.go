package mqtt

import (
	"reflect"
	"sync"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestManager_getClient(t *testing.T) {
	type args struct {
		clientID string
	}
	tests := []struct {
		name   string
		fields map[string]*client
		args   args
		want   *client
	}{
		{
			name: "exist",
			fields: map[string]*client{
				"client": &client{},
			},
			args: args{
				clientID: "client",
			},
			want: &client{},
		},
		{
			name:   "not exist",
			fields: map[string]*client{},
			args: args{
				clientID: "none",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_manager := &Manager{
				lock:      sync.RWMutex{},
				mapClient: tt.fields,
			}
			if got := _manager.getClient(tt.args.clientID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Manager.getClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_addClient(t *testing.T) {
	type fields struct {
		mapClient map[string]*client
	}
	type args struct {
		clientID  string
		username  string
		password  string
		mapBroker map[string]struct{}
		mapTopic  map[string]byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *client
	}{
		{
			name: "new",
			fields: fields{
				mapClient: map[string]*client{},
			},
			args: args{
				clientID:  "clientID",
				username:  "username",
				password:  "password",
				mapBroker: map[string]struct{}{},
				mapTopic:  map[string]byte{},
			},
			want: &client{
				username:  "username",
				password:  "password",
				mapBroker: map[string]struct{}{},
				mapTopic:  map[string]byte{},
			},
		},
		{
			name: "overwrite",
			fields: fields{
				mapClient: map[string]*client{
					"client": &client{
						username:  "username",
						password:  "password",
						mapBroker: map[string]struct{}{},
						mapTopic:  map[string]byte{},
					},
				},
			},
			args: args{
				clientID: "clientID",
				username: "usernameA",
				password: "passwordA",
				mapBroker: map[string]struct{}{
					"brokerA": struct{}{},
				},
				mapTopic: map[string]byte{
					"topicA": byte(2),
				},
			},
			want: &client{
				username: "usernameA",
				password: "passwordA",
				mapBroker: map[string]struct{}{
					"brokerA": struct{}{},
				},
				mapTopic: map[string]byte{
					"topicA": byte(2),
				},
			},
		},
		{
			name: "nil brokers & nil topics",
			fields: fields{
				mapClient: map[string]*client{},
			},
			args: args{
				clientID: "clientID",
				username: "username",
				password: "password",
			},
			want: &client{
				username: "username",
				password: "password",
				chMsg:    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_manager := &Manager{
				lock:      sync.RWMutex{},
				mapClient: tt.fields.mapClient,
			}
			got := _manager.addClient(tt.args.clientID, tt.args.username, tt.args.password, tt.args.mapBroker, tt.args.mapTopic)
			if got.username != tt.want.username {
				t.Errorf("Manager.addClient() -- username = %v, want %v", got.username, tt.want.username)
			}
			if got.password != tt.want.password {
				t.Errorf("Manager.addClient() -- password = %v, want %v", got.password, tt.want.password)
			}
			if !reflect.DeepEqual(got.mapBroker, tt.want.mapBroker) {
				t.Errorf("Manager.addClient() -- mapBroker = %v, want %v", got.mapBroker, tt.want.mapBroker)
			}
			if !reflect.DeepEqual(got.mapTopic, tt.want.mapTopic) {
				t.Errorf("Manager.addClient() -- mapTopic = %v, want %v", got.mapTopic, tt.want.mapTopic)
			}
			if got.chMsg == tt.want.chMsg {
				t.Errorf("Manager.addClient() -- chMsg = %v, want %v", got.mapTopic, tt.want.mapTopic)
			}
		})
	}
}

func TestManager_delClient(t *testing.T) {
	type fields struct {
		mapClient map[string]*client
	}
	type args struct {
		clientID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "exist",
			fields: fields{
				mapClient: map[string]*client{
					"client": &client{
						username:  "username",
						password:  "password",
						mapBroker: map[string]struct{}{},
						mapTopic:  map[string]byte{},
						chMsg:     make(chan mqtt.Message),
					},
				},
			},
			args: args{
				clientID: "client",
			},
			want: 0,
		},
		{
			name: "not exist",
			fields: fields{
				mapClient: map[string]*client{
					"client": &client{
						username:  "username",
						password:  "password",
						mapBroker: map[string]struct{}{},
						mapTopic:  map[string]byte{},
						chMsg:     make(chan mqtt.Message),
					},
				},
			},
			args: args{
				clientID: "no",
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_manager := &Manager{
				lock:      sync.RWMutex{},
				mapClient: tt.fields.mapClient,
			}

			_manager.delClient(tt.args.clientID)
			if len(_manager.mapClient) != tt.want {
				t.Errorf("Manager.delClient(), length of mapClient is %v, want %v", len(_manager.mapClient), tt.want)
			}
		})
	}
}
