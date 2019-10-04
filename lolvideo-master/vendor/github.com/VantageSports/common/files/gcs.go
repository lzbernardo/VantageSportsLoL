package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/VantageSports/common/credentials/google"
)

type GCSProvider struct {
	client      *storage.Client
	TimeoutSecs int
}

func NewGCSProvider(creds *google.Creds) (*GCSProvider, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithTokenSource(creds.Conf.TokenSource(ctx)))
	return &GCSProvider{client, 120}, err
}

func (g *GCSProvider) DownloadTo(src, localDst string) error {
	b, k, err := BucketKey(src)
	if err != nil {
		return err
	}

	r, err := g.client.Bucket(b).Object(k).NewReader(g.timeoutCtx())
	if err != nil {
		return err
	}

	f, err := os.Create(localDst)
	if err != nil {
		return err
	}

	return copyAndClose(f, r)
}

func (g *GCSProvider) UploadTo(localSrc, dst string, opts ...FileOption) error {
	b, k, err := BucketKey(dst)
	if err != nil {
		return err
	}

	f, err := os.Open(localSrc)
	if err != nil {
		return err
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	ctx := g.timeoutCtx()
	w := g.client.Bucket(b).Object(k).NewWriter(ctx)
	w.ContentType = detectContentType(localSrc, f)
	for _, opt := range opts {
		if err = opt.GCS(&w.ObjectAttrs); err != nil {
			return err
		}
	}

	copyResult := make(chan error, 1)
	go func() {
		copyResult <- copyAndClose(w, f)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-copyResult:
		// There's probably a different underlying problem for this... Occassionally, we
		// see uploads of 0 bytes with no errors.
		// Even the "Copy" operation returns the proper amount of bytes copied, but the destination is still empty
		// Try to read the file and see how big it is
		attrs, err := g.client.Bucket(b).Object(k).Attrs(g.timeoutCtx())
		if err != nil {
			return err
		}
		if stat.Size() > 0 && attrs.Size == 0 {
			return fmt.Errorf("expected %v bytes, uploaded %v bytes", stat.Size(), attrs.Size)
		}
		return res
	}
}

func (g *GCSProvider) AllowPublicRead(dst string) (string, error) {
	b, k, err := BucketKey(dst)
	if err != nil {
		return "", err
	}

	if err = g.client.Bucket(b).Object(k).ACL().Set(g.timeoutCtx(), storage.AllUsers, storage.RoleReader); err != nil {
		return "", err
	}
	return g.URLFor(dst)
}

func (g *GCSProvider) URLFor(src string) (string, error) {
	bucket, key, err := BucketKey(src)
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, key), err
}

func (g *GCSProvider) Copy(src, dst string, opts ...FileOption) error {
	srcBucket, srcKey, err := BucketKey(src)
	if err != nil {
		return err
	}
	dstBucket, dstKey, err := BucketKey(dst)
	if err != nil {
		return err
	}

	srcObj := g.client.Bucket(srcBucket).Object(srcKey)
	attrs, err := srcObj.Attrs(g.timeoutCtx())
	if err != nil {
		return err
	}
	if attrs == nil {
		attrs = &storage.ObjectAttrs{}
	}

	for _, opt := range opts {
		if err = opt.GCS(attrs); err != nil {
			return err
		}
	}

	dstObj := g.client.Bucket(dstBucket).Object(dstKey)
	copier := dstObj.CopierFrom(srcObj)
	copier.ObjectAttrs = *attrs

	_, err = copier.Run(g.timeoutCtx())
	return err
}

func (g *GCSProvider) Rename(src, dst string, opts ...FileOption) error {
	if err := g.Copy(src, dst, opts...); err != nil {
		return err
	}
	return g.Remove(src)
}

func (g *GCSProvider) Read(src string) ([]byte, error) {
	b, k, err := BucketKey(src)
	if err != nil {
		return nil, err
	}

	r, err := g.client.Bucket(b).Object(k).NewReader(g.timeoutCtx())
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r)
	r.Close()
	return data, err
}

func (g *GCSProvider) Remove(path string) error {
	b, k, err := BucketKey(path)
	if err != nil {
		return err
	}
	return g.client.Bucket(b).Object(k).Delete(g.timeoutCtx())
}

// HACK! Preserves old behavior since some things rely on that, while letting
// some clients increase. We should move this to a function-level param though.
var MaxListResults = 5000

func (g *GCSProvider) List(path string) ([]string, error) {
	b, k, err := BucketKey(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(k, "/") {
		k += "/"
	}

	it := g.client.Bucket(b).Objects(g.timeoutCtx(), &storage.Query{Prefix: k})

	keys := []string{}
	for {
		o, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		fullKey := fmt.Sprintf("gs://%s/%s", o.Bucket, o.Name)
		// Don't include the original directory in the result.
		if fullKey != path && fullKey != path+"/" {
			keys = append(keys, fullKey)
		}
		if len(keys) >= MaxListResults {
			break
		}
	}
	return keys, nil
}

func (g *GCSProvider) Exists(path string) (bool, error) {
	b, k, err := BucketKey(path)
	if err != nil {
		return false, err
	}
	_, err = g.client.Bucket(b).Object(k).Attrs(g.timeoutCtx())
	if err != nil && (err == storage.ErrObjectNotExist || err == storage.ErrBucketNotExist) {
		return false, nil
	}
	return err == nil, err
}

func copyAndClose(dst io.WriteCloser, src io.ReadCloser) error {
	_, copyErr := io.Copy(dst, src)
	dstErr := dst.Close()
	srcErr := src.Close()

	if copyErr != nil {
		return copyErr
	}
	if dstErr != nil {
		return dstErr
	}
	return srcErr
}

func (g *GCSProvider) timeoutCtx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(g.TimeoutSecs))
	return ctx
}
