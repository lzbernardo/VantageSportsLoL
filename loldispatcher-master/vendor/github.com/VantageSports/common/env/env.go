package env

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/VantageSports/common/log"
)

// Must calls log.Fatalln() if the key's value is empty..
func Must(key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	log.Fatal("missing Must() key:" + key)
	return "" // Dumb compiler
}

// MustInt exits the process if there is no value for key, or if that value
// cannot be parsed as an integer.
func MustInt(key string) int {
	val := Must(key)
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Fatal(fmt.Sprintf("MustInt for key: %s err: %v", key, err))
	}
	return intVal
}

// MustLoad exits the process if there is no file at the given path, or if the
// contents cannot be read as a string. If trim is true, the result is
// whitespace-trimmed.
//
// This is probably more of an "io" package thing, but we use it almost
// exclusively from main()s when parsing env flags, so it lives here for now.
func MustLoad(path string, trim bool) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(fmt.Sprintf("MustLoad path:%s err:%s", path, err))
	}
	if trim {
		return strings.TrimSpace(string(data))
	}
	return string(data)
}

// Or returns val if value for key is empty.
func Or(key, val string) string {
	if found := os.Getenv(key); found != "" {
		return found
	}
	return val
}

// SmartString requires a value for the given key in the form "<prefix>:<value>"
// using the prefix to determine how to interpret the value. For prefix values
// of:
//  "string": returns <value> unmodified
//  "file": assumes <value> is a file path and returns the contents as a string
//  "file_trim": like "file", but trims whitespace from returned string
//
// This is typically useful when we want to load values directly from strings
// in some contexts (development) but from files in other contexts (production)
// but not need additional logic to choose different keys based on environment.
func SmartString(key string) string {
	taggedVal := Must(key)

	parts := strings.SplitN(taggedVal, ":", 2)
	if len(parts) != 2 {
		log.Fatal(fmt.Sprintf("SmartString requires format <tag>:<val>, found %s %s", taggedVal, parts))
	}
	tag, val := parts[0], parts[1]

	// choose an env key unlikely to be used :)
	tmpKey := "smart_string_2010_0123456789"
	os.Setenv(tmpKey, val)

	switch tag {
	case "string":
		return Must(tmpKey)
	case "file":
		return MustLoad(val, false)
	case "file_trim":
		return MustLoad(val, true)
	default:
		log.Fatal("unrecognized SmartString tag:" + tag)
		return ""
	}
}
