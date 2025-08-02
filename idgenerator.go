package elorm

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

const refStringLength = 12 // length for reference string, enough for a century
var refBaseTime = time.Date(2025, time.June, 23, 0, 0, 0, 0, time.Local).UTC()
var lastRefValue int64
var lastRefValueLock sync.Mutex

// NewRef generates a new unique ID/URL for entities and other purposes.
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
	for len(res) < refStringLength {
		res = fmt.Sprintf("0%s", res)
	}
	return res
}
