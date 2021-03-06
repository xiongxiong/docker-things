package mqtt

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
	brok := "tcp://localhost:1883"
	topi := "myTopic"

	FDescribe("subscribe and unsubscribe", func() {
		It("one topic", func() {

			By("subscribe")
			err := SubBrokerTopic(brok, topi, nil)
			Ω(err).ToNot(HaveOccurred(), "cannot subscribe")
			Ω(_Global.mapConn).To(HaveLen(1))
			_broker := _Global.mapConn[brok]
			Ω(_broker.client).ToNot(BeZero())
			Ω(_broker.chMsg).ToNot(BeZero())
			Ω(_broker.chQuit).ToNot(BeZero())
			Ω(_broker.mapTopic).To(And(
				Not(BeZero()),
				HaveLen(1),
				MatchAllKeys(Keys{
					topi: Not(BeZero()),
				})))

			By("unsubscribe")
			err = UnSubBrokerTopic(brok, topi)
			Ω(err).ToNot(HaveOccurred(), "cannot unsubscribe")
			Ω(_broker.chQuit).To(BeClosed())
			Eventually(func() int {
				return len(_Global.mapConn)
			}).Should(Equal(0))
		})
	})

	Describe("resubscribe", func() {

	})

	Describe("publish and receive", func() {
		var chMsg chan string
		var _broker *broker

		BeforeEach(func() {
			chMsg = make(chan string)

			err := SubBrokerTopic(brok, topi, func(topic, message string) {
				chMsg <- message
			})
			Ω(err).ToNot(HaveOccurred(), "cannot subscribe")

			_broker = _Global.mapConn[brok]
			Ω(_broker).ToNot(BeNil())
		})

		AfterEach(func() {
			err := UnSubBrokerTopic(brok, topi)
			Ω(err).ToNot(HaveOccurred(), "cannot unsubscribe")

			close(chMsg)
		})

		It("should publish and receive message success", func() {
			By("publish")
			token := _broker.client.Publish(topi, byte(2), false, "hello")
			token.Wait()
			Ω(token.Error()).ToNot(HaveOccurred(), "cannot publish")

			By("receive")
			Ω(<-chMsg).To(Equal("hello"))
		})
	})

	Describe("mixture", func() {

	})
})
