package interfaces

import "time"

type IClockProxy interface {
	Now() time.Time
}
