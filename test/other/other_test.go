package other

import (
	"sync/atomic"
	"testing"
)

func TestAtomicAdd(t *testing.T) {
	var c int32 = 0
	atomic.AddInt32(&c, 1)
	if c != 1 {
		t.Error("atomic not modify the origin value")
	}
	t.Logf("c is %d", c)
}

func TestChannel(t *testing.T) {

}
