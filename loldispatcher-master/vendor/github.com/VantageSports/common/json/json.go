package json

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/VantageSports/common/files/util"
)

// Compress marshals the interface and then gzips the data.
func Compress(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return util.GzipBytes(data)
}

func DecodeIf(r io.Reader, byteLimit int64, v interface{}) error {
	if byteLimit > 0 {
		r = io.LimitReader(r, byteLimit)
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func DecodeLimit(r io.Reader, byteLimit int64, v interface{}) error {
	if byteLimit > 0 {
		r = io.LimitReader(r, byteLimit)
	}
	return json.NewDecoder(r).Decode(v)
}

func DecodeFile(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func Write(filename string, v interface{}, perm os.FileMode) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, perm)
}

func WriteIndent(filename string, v interface{}, perm os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, perm)
}
