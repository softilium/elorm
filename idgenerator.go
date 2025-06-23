package elorm

import (
	"strconv"
	"sync"
	"time"
)

var lastRefValue int64
var lastRefValueLock sync.Mutex

// Reference time constant - only needs to be created once
var refBaseTime = time.Date(2025, time.June, 23, 0, 0, 0, 0, time.Local).UTC()

func NewRef() string {
	lastRefValueLock.Lock()
	defer lastRefValueLock.Unlock()

	now := time.Now().UTC().Sub(refBaseTime).Nanoseconds()
	if now > lastRefValue {
		lastRefValue = now
	} else {
		lastRefValue++
	}
	res := strconv.FormatInt(lastRefValue, 36)
	for len(res) < 12 {
		res = "0" + res
	}
	return res
}
