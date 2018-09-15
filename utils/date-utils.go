package utils

import (
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
	// ISOTime - iso time format
	ISOTime string = "15:04:05"
	// ISOTimeMS - iso time format with miliseconds
	ISOTimeMS string = "15:04:05.000"
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
	RSSDateTime string = "Mon, 02 Jan 2006 15:04:05 Z07:00"
	// RSSDateTime1 - rss date time format 1
	RSSDateTime1 string = "Mon, _2 Jan 2006 15:04:05 Z07:00"
	// RSSDateTime2 - rss date time format 2
	RSSDateTime2 string = "Mon, 02 Jan 2006 15:04:05 Z0700"
	// RSSDateTime3 - rss date time format 3
	RSSDateTime3 string = "Mon, _2 Jan 2006 15:04:05 Z0700"
	// RSSDateTimeTZ - rss date time format with named timezone
	RSSDateTimeTZ string = "Mon, 02 Jan 2006 15:04:05 MST"
	// RSSDateTimeTZ1 - rss date time format with named timezone 1
	RSSDateTimeTZ1 string = "Mon, _2 Jan 2006 15:04:05 MST"
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
	case UTCDate:
		return val.UTC().Format(ISODate)
	case UTCDateTime:
		return val.UTC().Format(ISODateTimeZ)
	case UTCDateTimestamp:
		return val.UTC().Format(ISODateTimestampZ)
	default:
		return val.Format(format)
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
	default:
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return time.Now(), err
		}

		t, err := time.ParseInLocation(format, sval, loc)
		if err != nil {
			return time.Now(), err
		}
		return t, nil
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

// ParseRSSDate - try to parse RSS date in multiple formats
func ParseRSSDate(sdate string) (time.Time, error) {
	var err error
	var dt time.Time

	formats := []string{
		RSSDateTimeTZ,
		RSSDateTimeTZ1,
		RSSDateTime,
		RSSDateTime1,
		RSSDateTime2,
		RSSDateTime3,
		ISODateTime,
		ISODateTimeZ,
	}

	for _, format := range formats {
		dt, err = String2date(sdate, format)
		if err == nil {
			break
		}
	}

	return dt.UTC(), err
}
