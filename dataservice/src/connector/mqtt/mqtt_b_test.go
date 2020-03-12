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

	cmd = exec.Command("sh", "-c", "waitforit -host=localhost -port=1883 -timeout=20 -debug")
	err = cmd.Run()
	Ω(err).NotTo(HaveOccurred(), "wait for mosquitto")
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

		By("publish message 1 on myTopic")
		cmd := exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopic' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("receive message 1 on myTopic")
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCount)
		}).Should(Equal(int32(1)))

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

		By("publish message 1 on myTopicNew")
		cmd = exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopicNew' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("receive message 1 on myTopicNew")
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCount)
		}).Should(Equal(int32(1)))
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCountNew)
		}).Should(Equal(int32(1)))

		By("publish message 2 on myTopic")
		cmd = exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopic' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("receive message 2 on myTopic, should receive only once")
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCount)
		}).Should(Equal(int32(2)))
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCountNew)
		}).Should(Equal(int32(1)))

		By("unsubscribe")
		manager.UnSubscribe(clientID)

		By("publish message 3 on myTopic")
		cmd = exec.Command("sh", "-c", `docker exec mosquitto sh -c "mosquitto_pub -t 'myTopic' -m 'hello'"`)
		err = cmd.Run()
		Ω(err).NotTo(HaveOccurred(), "mosquitto cannot publish")

		By("does not receive message after unsubscription")
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCount)
		}).Should(Equal(int32(2)))
		Eventually(func() int32 {
			return atomic.LoadInt32(&msgCountNew)
		}).Should(Equal(int32(1)))
	})

	// TODO benchmark
})
