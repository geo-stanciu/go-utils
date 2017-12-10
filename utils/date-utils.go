package utils

import (
	"fmt"
	"net/http"
	"time"
)

const (
	ISODate           string = "2006-01-02"
	ISODateTime       string = "2006-01-02 15:04:05"
	ISODateTimestamp  string = "2006-01-02 15:04:05.000"
	ISODateTimeZ      string = "2006-01-02 15:04:05Z07:00"
	ISODateTimestampZ string = "2006-01-02 15:04:05.000Z07:00"
	DMY               string = "02/01/2006"
	DMYTime           string = "02/01/2006 15:04:05"
	UTCDateTime       string = "UTC"
	UTCDateTimestamp  string = "UTCTimestamp"
	DateOffset        string = "Z07:00"
)

func IsISODate(sval string) bool {
	_, err := String2date(sval, ISODate)

	if err != nil {
		return false
	}

	return true
}

func IsISODateTime(sval string) bool {
	_, err := String2date(sval, ISODateTime)

	if err != nil {
		return false
	}

	return true
}

func DateFromISODateTime(sval string) (time.Time, error) {
	return String2date(sval, ISODateTime)
}

func Date2string(val time.Time, format string) string {
	switch format {
	case ISODate, ISODateTime, ISODateTimestamp, ISODateTimeZ, ISODateTimestampZ, DMY, DMYTime:
		return val.Format(format)
	case UTCDateTime:
		return val.UTC().Format(ISODateTimeZ)
	case UTCDateTimestamp:
		return val.UTC().Format(ISODateTimestampZ)
	default:
		return ""
	}

}

func String2dateNoErr(sval string, format string) time.Time {
	dt, err := String2date(sval, format)
	if err != nil {
		panic(err)
	}
	return dt
}

func String2date(sval string, format string) (time.Time, error) {
	switch format {
	case ISODate, ISODateTime, ISODateTimestamp, ISODateTimeZ, ISODateTimestampZ, DMY, DMYTime, DateOffset:
		loc, err := time.LoadLocation("Local")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(format, sval, loc)
		if err != nil {
			return time.Now(), err
		}
		return t, nil
	case UTCDateTime:
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(ISODateTime, sval, loc)
		if err != nil {
			return time.Now(), err
		}
		return t, nil
	case UTCDateTimestamp:
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(ISODateTimestamp, sval, loc)
		if err != nil {
			return time.Now(), err
		}
		return t, nil
	default:
		return time.Now(), fmt.Errorf("Unknown datetime format \"%s\"", format)
	}
}

func Server2ClientDmy(r *http.Request, serverTime time.Time) string {
	t := Server2ClientLocal(r, serverTime)
	return Date2string(t, DMY)
}

func Server2ClientDmyTime(r *http.Request, serverTime time.Time) string {
	t := Server2ClientLocal(r, serverTime)
	return Date2string(t, DMYTime)
}

func Server2ClientLocal(r *http.Request, serverTime time.Time) time.Time {
	timeOffset := 0

	cookie, err := r.Cookie("time_zone_offset")
	if err != nil && err != http.ErrNoCookie {
		return serverTime.UTC()
	} else if err == http.ErrNoCookie {
		timeOffset = 0
	} else {
		timeOffset = String2int(cookie.Value)
	}

	return serverTime.UTC().Add(time.Duration(-1*timeOffset) * time.Minute)
}
