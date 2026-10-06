package utilities

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const republicOfChinaYearOffset = 1911

var taipeiLocation = time.FixedZone("Asia/Taipei", 8*60*60)

type RepublicOfChinaDateParser struct{}

func NewRepublicOfChinaDateParser() *RepublicOfChinaDateParser {
	return &RepublicOfChinaDateParser{}
}

// Taiwan exchanges publish dates in the Republic of China calendar, e.g. 1151005 is 2026-10-05
func (republicOfChinaDateParser *RepublicOfChinaDateParser) ParseTaipeiMidnight(rawDate string) (time.Time, error) {
	trimmedDate := strings.TrimSpace(rawDate)
	if len(trimmedDate) < 7 {
		return time.Time{}, fmt.Errorf("unexpected Republic of China date %q", rawDate)
	}
	republicOfChinaYear, err := strconv.Atoi(trimmedDate[:len(trimmedDate)-4])
	if err != nil {
		return time.Time{}, fmt.Errorf("unexpected Republic of China date %q: %w", rawDate, err)
	}
	return time.ParseInLocation("20060102", strconv.Itoa(republicOfChinaYear+republicOfChinaYearOffset)+trimmedDate[len(trimmedDate)-4:], taipeiLocation)
}
