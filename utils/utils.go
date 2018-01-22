package utils

import (
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

// GetUserHomeDir - Get User Home Dir
func GetUserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

// InTimeSpan - Checks if date is in time interval
func InTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

// InvokeMethodByName - Invokes Method By Name
func InvokeMethodByName(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}

	return reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

// String2int - String to int
func String2int(sval string) int {
	val, err := strconv.Atoi(sval)

	if err != nil {
		return 0
	}

	return val
}

// ContainsRepeatingGroups - Contains Repeating Groups
func ContainsRepeatingGroups(str string) bool {
	groupSize := 2
	length := len(str) - 1

	for i := groupSize; i < length; i++ {
		if testRepeatingGroups(str, length, i) {
			return true
		}
	}

	return false
}

func testRepeatingGroups(str string, length int, groupSize int) bool {
	for i := 0; i < length; i = i + groupSize {
		for j := i + groupSize; j < length-groupSize; j = j + groupSize {
			if str[i:i+groupSize] == str[j:j+groupSize] {
				return true
			}
		}
	}

	return false
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// GetMinGreaterThanZero - gets the first of two numbers
func GetMinGreaterThanZero(a, b int) int {
	if a > 0 && b > 0 {
		if a <= b {
			return a
		}
		return b
	} else if a > 0 {
		return a
	} else if b > 0 {
		return b
	}

	return -1
}

// IsWhiteSpace checks if string is space, tab or enter
func IsWhiteSpace(s string) bool {
	if s == " " || s == "\t" || s == "\r" || s == "\n" {
		return true
	}
	return false
}
