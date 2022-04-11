package utils

import (
	"os"
	"time"
)

const (
	TimeFormat      = "2006-01-02T15:04:05Z"
	DefaultTimeZone = "Asia/Shanghai"
)

func GetLocalTime() string {
	loc := GetLocation()
	if loc == nil {
		time.Now().Format(TimeFormat)
	}

	return time.Now().In(loc).Format(TimeFormat)
}

func GetLocation() *time.Location {
	zone := os.Getenv("TZ")

	if zone == "" {
		zone = DefaultTimeZone
	}

	loc, _ := time.LoadLocation(zone)

	return loc
}
