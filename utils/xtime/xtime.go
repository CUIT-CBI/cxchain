package xtime

import "time"

// second
func Now() uint64 {
	return uint64(time.Now().Unix())
}

// ms
func NowMS() uint64 {
	return uint64(time.Now().Nanosecond() / 1e6)
}
