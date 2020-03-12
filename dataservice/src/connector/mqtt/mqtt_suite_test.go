package mqtt_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMQTT(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mqtt suite")
}
