package system

import "time"

type SystemClockProxy struct{}

func NewSystemClockProxy() *SystemClockProxy {
	return &SystemClockProxy{}
}

func (systemClockProxy *SystemClockProxy) Now() time.Time {
	return time.Now()
}
