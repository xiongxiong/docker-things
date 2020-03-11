package main

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	cmd := exec.Command("sh", "-c", "docker-compose -f ../../ci/docker-compose.yml up -d")
	err := cmd.Run()
	Ω(err).NotTo(HaveOccurred(), "docker-compose cannot start")
})

var _ = AfterSuite(func() {
	cmd := exec.Command("sh", "-c", "docker-compose down")
	err := cmd.Run()
	Ω(err).NotTo(HaveOccurred(), "docker-compose cannot stop")
})

var _ = Describe("main", func() {
	It("subscribe should ok", func() {

	})
	It("unsubscribe should ok", func() {

	})
})
