package mosquitto

import (
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
	// 	"github.com/onsi/ginkgo/ginkgo"
	// "github.com/onsi/gomega/..."
)

func TestMain(m *testing.M) {
	mosStart := exec.Command("sh", "-c", "docker run -d --rm --name mosquitto eclipse-mosquitto")
	if err := mosStart.Run(); err != nil {
		log.Println(err.Error())
	}
	time.Sleep(30 * time.Second)
	os.Exit(m.Run())
	mosStop := exec.Command("sh", "-c", "docker stop mosquitto")
	if err := mosStop.Run(); err != nil {
		log.Println(err.Error())
	}
}
