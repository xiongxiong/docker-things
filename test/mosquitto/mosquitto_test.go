package mosquitto

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestMain(m *testing.M) {
	mosStart := exec.Command("sh", "-c", "docker run --detach --rm --name mosquitto --publish 1883:1883 eclipse-mosquitto")
	if err := mosStart.Run(); err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("mosquitto started")
	mID := m.Run()
	mosStop := exec.Command("sh", "-c", "docker stop mosquitto")
	if err := mosStop.Run(); err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("mosquitto stoped")
	}
	os.Exit(mID)
}

func TestMos(t *testing.T) {
	// t.Skip()

	msgCount := 0

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		msgCount++
	})
	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot connect mosquitto -- %s", token.Error().Error())
	}
	filters := map[string]byte{"aaa": byte(2)}
	if token := client.SubscribeMultiple(filters, nil); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot subscribe -- %s", token.Error().Error())
	}

	if token := client.Publish("aaa", byte(2), true, "aaa1"); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot publish -- %s", token.Error().Error())
	}
	time.Sleep(time.Second)
	if msgCount != 1 {
		t.Fatalf("cannot receive message aaa")
	}

	filters["bbb"] = byte(2)
	if token := client.Publish("bbb", byte(2), true, "bbb"); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot publish -- %s", token.Error().Error())
	}
	time.Sleep(time.Second)
	if msgCount != 1 {
		t.Fatalf("should not receive message bbb")
	}

	if token := client.Subscribe("aaa", byte(2), nil); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot subscribe -- %s", token.Error().Error())
	}
	if token := client.Publish("aaa", byte(2), true, "aaa2"); token.Wait() && token.Error() != nil {
		t.Fatalf("cannot publish -- %s", token.Error().Error())
	}
	time.Sleep(time.Second)
	if msgCount != 2 {
		t.Fatalf("should receive message twice -- %d", msgCount)
	}

	client.Disconnect(1000)
}
