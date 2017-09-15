package utils

import (
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

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

func InTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func InvokeMethodByName(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}

	return reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

func String2int(sval string) int {
	val, err := strconv.Atoi(sval)

	if err != nil {
		return 0
	}

	return val
}

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
