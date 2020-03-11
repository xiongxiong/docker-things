package mqtt_test

import (
	mqttC "dataservice/connector/mqtt"
	"os/exec"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	cmd := exec.Command("sh", "-c", "docker run --detach --rm --name mosquitto --publish 1883:1883 eclipse-mosquitto")
	err := cmd.Run()
	Ω(err).NotTo(HaveOccurred(), "mosquitto cannot start")
})

var _ = AfterSuite(func() {
	cmd := exec.Command("sh", "-c", "docker stop mosquitto")
	err := cmd.Run()
	Ω(err).NotTo(HaveOccurred(), "mosquitto cannot stop")
})

var _ = Describe("mqtt", func() {
	manager := mqttC.NewManager()

	It("normal flow", func() {
		clientID := "myClientID"
		username := ""
		password := ""
		mapBroker := map[string]struct{}{
			"tcp://localhost:1883": struct{}{},
		}
		mapTopic := map[string]byte{
			"myTopic": byte(2),
		}
		var msgCount int32
		msgProc := func(clientID string, msg mqtt.Message) {
			atomic.AddInt32(&msgCount, 1)
		}

		By("client subscribe")
		err := manager.Subscribe(clientID, username, password, mapBroker, mapTopic, msgProc)
		Ω(err).ToNot(HaveOccurred(), "cannot subscribe")

		By("publish message 1")
		cmd := exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopic' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("receive message 1")
		Eventually(func() int {
			return int(msgCount)
		}).Should(Equal(1))

		By("same client resubscribe")
		mapTopic = map[string]byte{
			"myTopic":    byte(2),
			"myTopicNew": byte(2),
		}
		var msgCountNew int32
		msgProc = func(clientID string, msg mqtt.Message) {
			switch msg.Topic() {
			case "myTopic":
				atomic.AddInt32(&msgCount, 1)
			case "myTopicNew":
				atomic.AddInt32(&msgCountNew, 1)
			}
		}
		err = manager.Subscribe(clientID, username, password, mapBroker, mapTopic, msgProc)
		Ω(err).ToNot(HaveOccurred(), "cannot subscribe")

		By("publish message 2")
		cmd = exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopicNew' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("receive message 2")
		Eventually(func() int {
			return int(msgCount)
		}).Should(Equal(1))
		Eventually(func() int {
			return int(msgCountNew)
		}).Should(Equal(1))

		By("publish message 3")
		cmd = exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopic' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("unsubscribe and message all processed before client close")
		manager.UnSubscribe(clientID)
		Eventually(func() int {
			return int(msgCount)
		}).Should(Equal(2))
		Eventually(func() int {
			return int(msgCountNew)
		}).Should(Equal(1))
	})

	// TODO benchmark
})
