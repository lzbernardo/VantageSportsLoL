package files

import (
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/service/s3"
)

// FileOption sets properties of remote objects at write-time.
type FileOption interface {
	GCS(attrs *storage.ObjectAttrs) error
	S3(v interface{}) error
}

// ContentType is a FileOption that sets the Content-Type for an object.
type ContentType string

func (c ContentType) GCS(attrs *storage.ObjectAttrs) error {
	attrs.ContentType = string(c)
	return nil
}

func (c ContentType) S3(v interface{}) error {
	val := string(c)
	switch t := v.(type) {
	case *s3.PutObjectInput:
		t.ContentType = &val
	case *s3.CopyObjectInput:
		t.ContentType = &val
	default:
		return fmt.Errorf("unknown type %T", v)
	}
	return nil
}

// ContentEncoding is a FileOption that sets the Content-Encoding for an object.
type ContentEncoding string

func (c ContentEncoding) GCS(attrs *storage.ObjectAttrs) error {
	attrs.ContentEncoding = string(c)
	return nil
}

func (c ContentEncoding) S3(v interface{}) error {
	val := string(c)
	switch t := v.(type) {
	case *s3.PutObjectInput:
		t.ContentEncoding = &val
	case *s3.CopyObjectInput:
		t.ContentEncoding = &val
	default:
		return fmt.Errorf("unknown type %T", v)
	}
	return nil
}
