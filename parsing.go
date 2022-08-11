package main

import (
	"fmt"
	"strconv"
)

// Temperatures are expressed as 2bytes hexadecimal string, eg: "TEMP=010a"
// 0x010a --(base 2)--> 0000 0001 0000 1010
//                      |\                /
//                      | \              /
//                   sign   value on 15 bits
//        (positive if 0,   to divide by 10
//         negative if 1)   -> 266 / 10 = 26.6
func parseTemp(s string) (float64, error) {
	var sign int64
	sign = 1

	if len(s) != 4 {
		return 0.0, fmt.Errorf("Couldn't parse temperature: value must be a string of exactly 4 characters")
	}

	intVal, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return 0.0, fmt.Errorf("Couldn't parse hex value %s of kind TEMP: %v", s, err)
	}

	if intVal&0x8000 == 0x8000 {
		sign = -1
	}

	// Set highest bit to zero since we already looked at the sign
	return float64(sign*(intVal&0x7FFF)) / 10, nil
}

// Converts a string into a positive integer on a given base then casts it into
// a float64
// This is used for all the simple values
func strToFloat(s string, base int) (float64, error) {
	u, err := strconv.ParseUint(s, base, 16)
	if err != nil {
		return 0.0, err
	}
	return float64(u), nil
}

// Parses a string as a certain value given the kind of data that is represented
// by the value
func parseValue(value string, kind string) (float64, error) {
	var (
		f   float64
		err error
	)

	switch kind {
	case "battery":
		switch value {
		case "OK":
			f = 1.0
		case "LOW":
			f = 0.0
		default:
			return 0.0, fmt.Errorf("Unknown value %s for the kind %s", value, kind)
		}
	case "onoff":
		switch value {
		case "ON":
			f = 1.0
		case "OFF":
			f = 0.0
		default:
			return 0.0, fmt.Errorf("Unknown value %s for the kind %s", value, kind)
		}
	case "hex":
		f, err = strToFloat(value, 16)
		if err != nil {
			return 0.0, fmt.Errorf("Couldn't parse hex value %s of kind %s: %v", value, kind, err)
		}
	case "hex_div10":
		f, err = strToFloat(value, 16)
		if err != nil {
			return 0.0, fmt.Errorf("Couldn't parse hex value %s of kind %s: %v", value, kind, err)
		}
		f /= 10
	case "int":
		f, err = strToFloat(value, 10)
		if err != nil {
			return 0.0, fmt.Errorf("Couldn't parse hex value %s of kind %s: %v", value, kind, err)
		}
	case "temp":
		f, err = parseTemp(value)
	default:
		return 0.0, fmt.Errorf("Unknown kind %s", kind)
	}
	return f, nil
}
