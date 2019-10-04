package util

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
)

// GzipFile reads src, gzips the contents and writes a new file to dst.
func GzipFile(src, dst string, perm os.FileMode) error {
	in, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	compressed, err := GzipBytes(in)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, compressed, perm)
}

// GzipBytes returns a byte slice representing the gzipped input slice.
func GzipBytes(in []byte) ([]byte, error) {
	out := bytes.Buffer{}
	w := gzip.NewWriter(&out)
	if _, err := w.Write(in); err != nil {
		return nil, err
	}

	err := w.Close()
	return out.Bytes(), err
}

// GunzipBytes returns a byte slice representing the unzipped input slice.
func GunzipBytes(in []byte) ([]byte, error) {
	zipped := bytes.NewReader(in)
	r, err := gzip.NewReader(zipped)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}
