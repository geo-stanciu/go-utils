package utils

import (
	"fmt"
	"net/http"
	"time"
)

const (
	// ISODate - iso date format
	ISODate string = "2006-01-02"
	// ISODateTime - iso date time format
	ISODateTime string = "2006-01-02 15:04:05"
	// ISODateTimestamp - iso timestamp format
	ISODateTimestamp string = "2006-01-02 15:04:05.000"
	// ISODateTimeZ - iso datetime with timezone format
	ISODateTimeZ string = "2006-01-02 15:04:05Z07:00"
	// ISODateTimestampZ - iso timestamp with timezone format
	ISODateTimestampZ string = "2006-01-02 15:04:05.000Z07:00"
	// DMY - dd/MM/yyyy
	DMY string = "02/01/2006"
	// DMYTime - dd/MM/yyyy HH:m:ss
	DMYTime string = "02/01/2006 15:04:05"
	// UTCDate - date at midnight UTC
	UTCDate string = "UTCDate"
	// UTCDateTime - ISODateTime at UTC
	UTCDateTime string = "UTC"
	// UTCDateTimestamp - ISODateTimestamp at UTC
	UTCDateTimestamp string = "UTCTimestamp"
	// DateOffset - time zone offset
	DateOffset string = "Z07:00"
	// RSSDateTime - rss date time format
	RSSDateTime string = "Mon, 02 Jan 2006 15:04:05 MST"
)

// IsISODate - checks if is in iso date format
func IsISODate(sval string) bool {
	_, err := String2date(sval, ISODate)

	if err != nil {
		return false
	}

	return true
}

// IsISODateTime - checks if is in iso datetime format
func IsISODateTime(sval string) bool {
	_, err := String2date(sval, ISODateTime)

	if err != nil {
		return false
	}

	return true
}

// DateFromISODateTime - Date From ISODateTime
func DateFromISODateTime(sval string) (time.Time, error) {
	return String2date(sval, ISODateTime)
}

// Date2string - Date to string
func Date2string(val time.Time, format string) string {
	switch format {
	case ISODate, ISODateTime, ISODateTimestamp, ISODateTimeZ, ISODateTimestampZ, DMY, DMYTime:
		return val.Format(format)
	case UTCDate:
		return val.UTC().Format(ISODate)
	case UTCDateTime:
		return val.UTC().Format(ISODateTimeZ)
	case UTCDateTimestamp:
		return val.UTC().Format(ISODateTimestampZ)
	case RSSDateTime:
		return val.UTC().Format(RSSDateTime)
	default:
		return ""
	}

}

// String2dateNoErr - String to date NoErrCheck
func String2dateNoErr(sval string, format string) time.Time {
	dt, err := String2date(sval, format)
	if err != nil {
		panic(err)
	}
	return dt
}

// String2date - String to date
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
	case UTCDate:
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(ISODate, sval, loc)
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
	case RSSDateTime:
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(RSSDateTime, sval, loc)
		if err != nil {
			return time.Now(), err
		}
		return t, nil
	default:
		return time.Now(), fmt.Errorf("Unknown datetime format \"%s\"", format)
	}
}

// Server2ClientDmy - Server2ClientDmy
func Server2ClientDmy(r *http.Request, serverTime time.Time) string {
	t := Server2ClientLocal(r, serverTime)
	return Date2string(t, DMY)
}

// Server2ClientDmyTime - Server2ClientDmyTime
func Server2ClientDmyTime(r *http.Request, serverTime time.Time) string {
	t := Server2ClientLocal(r, serverTime)
	return Date2string(t, DMYTime)
}

// Server2ClientLocal - Server2ClientLocal
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
