package locks

import (
	_ "fmt"
	"testing"
)

func TestLock(t *testing.T) {

	unlocker := KeyLocker("foo1").Lock()
	if unlocker == nil {
		t.Errorf("Didn't obtain lock")
	}

	unlocker2 := KeyLocker("foo1").Lock()
	if unlocker2 != nil {
		t.Error("Did obtain lock")
	}

	unlocker.Unlock()
	unlocker = KeyLocker("foo1").Lock()
	if unlocker == nil {
		t.Errorf("Didn't obtain lock")
	}

}
