//go:build windows

package refresh

import (
	"sync"
	"testing"
	"time"
)

func TestScheduleIE4UInitIsThrottled(t *testing.T) {
	mu.Lock()
	lastInit = time.Now()
	mu.Unlock()

	start := time.Now()
	scheduleIE4UInit()
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("scheduleIE4UInit() did not return quickly: %v", elapsed)
	}

	mu.Lock()
	got := lastInit
	mu.Unlock()
	if time.Since(got) > time.Second {
		t.Fatal("throttled call moved lastInit")
	}
}

func TestFolderDoesNotPanic(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Folder(`C:\Windows`)
	}()
	wg.Wait()
}
