package mqtt

import (
	"reflect"
	"sync"
	"testing"
)

func TestClient(t *testing.T) {
	m := NewManager()
	id, name, pass := "id", "name", "pass"

	t.Log("add client")
	m.addClient(id, name, pass, nil, nil)
	if len(m.mapClient) != 1 {
		t.Error("add client error")
	}

	t.Log("add duplicate client")

	t.Log("del client")
}

func TestManager_getClient(t *testing.T) {
	type fields struct {
		lock      sync.RWMutex
		mapClient map[string]*client
	}
	type args struct {
		clientID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_manager := &Manager{
				lock:      tt.fields.lock,
				mapClient: tt.fields.mapClient,
			}
			if got := _manager.getClient(tt.args.clientID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Manager.getClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
