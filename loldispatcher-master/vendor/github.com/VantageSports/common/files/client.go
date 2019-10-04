// The files library is a thin wrapper around existing file systems. The idea
// is to have a single interface from which to do basic file manipulations
// irrespective of the storage technology involved (local filesystems, remote
// storage providers, etc.)

package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	managersByPrefix map[string]FileManager
	remotesByPrefix  map[string]Remote
}

func NewClient(localTmpDir string) (*Client, error) {
	localProvider, err := NewLocalProvider(localTmpDir)
	if err != nil {
		return nil, err
	}

	c := &Client{
		managersByPrefix: map[string]FileManager{},
		remotesByPrefix:  map[string]Remote{},
	}
	c.Register("/", localProvider, nil) // Because fuck windows?
	return c, nil
}

// Register associates the path prefix with a FileManager and (optional)
// Remote.
func (c *Client) Register(prefix string, fm FileManager, r Remote) error {
	if fm == nil {
		return fmt.Errorf("file manager cannot be nil")
	}

	if _, found := c.managersByPrefix[prefix]; found {
		return fmt.Errorf("prefix already registered for another file manager:", prefix)
	}
	c.managersByPrefix[prefix] = fm

	if r != nil {
		if _, found := c.remotesByPrefix[prefix]; found {
			return fmt.Errorf("remote already registered for prefix:" + prefix)
		}
		c.remotesByPrefix[prefix] = r
	}

	return nil
}

func (c *Client) ManagerFor(path string) FileManager {
	for prefix, fm := range c.managersByPrefix {
		if strings.HasPrefix(path, prefix) {
			return fm
		}
	}
	return nil
}

func (c *Client) RemoteFor(path string) Remote {
	for prefix, r := range c.remotesByPrefix {
		if strings.HasPrefix(path, prefix) {
			return r
		}
	}
	return nil
}

// Move moves a file/object from src to dst. If src and dst belong to different
// remotes, the file is downloaded locally and then uploaded to dst, and then
// the src copy is deleted.
func (c *Client) Move(src, dst string, opts ...FileOption) error {
	srcMgr, dstMgr := c.ManagerFor(src), c.ManagerFor(dst)

	if srcMgr == dstMgr {
		return srcMgr.Rename(src, dst)
	}
	return c.downloadUpload(src, dst, true, opts...)
}

// Copies a file/object from src to dst. If src and dst belong to different
// remotes, the file is downloaded locally and then uploaded to dst.
func (c *Client) Copy(src, dst string, opts ...FileOption) error {
	srcMgr, dstMgr := c.ManagerFor(src), c.ManagerFor(dst)

	if srcMgr == dstMgr {
		return srcMgr.Copy(src, dst, opts...)
	}
	return c.downloadUpload(src, dst, false, opts...)
}

// downloadUpload, surprisingly, downloads the file/object locally, then
// uploads it to the dst service (if dst is not local), then optionally deletes
// src. One of src or dst is expected to be a remote.
func (c *Client) downloadUpload(src, dst string, deleteSrc bool, opts ...FileOption) error {
	// Download to a temporary file.
	localPath := src
	srcRemote := c.RemoteFor(src)
	if srcRemote != nil {
		localPath = LocalPath(src)
		defer os.Remove(localPath)
		if err := srcRemote.DownloadTo(src, localPath); err != nil {
			return err
		}
	}

	// At this point, localPath is guaranteed to be the file that we want to
	// place at dst. If dst is a remote, then we'll upload it there, otherwise,
	// we'll just rename localPath.
	var err error
	if dstRemote := c.RemoteFor(dst); dstRemote != nil {
		err = dstRemote.UploadTo(localPath, dst, opts...)
	} else {
		err = c.ManagerFor(localPath).Rename(localPath, dst, opts...)
	}

	if err == nil && deleteSrc && src != dst {
		err = c.ManagerFor(src).Remove(src)
	}
	return err
}

func (c *Client) List(dir string) ([]string, error) {
	return c.ManagerFor(dir).List(dir)
}

func (c *Client) Read(src string) ([]byte, error) {
	return c.ManagerFor(src).Read(src)
}

// Returns true if the OBJECT/FILE referenced by src is valid.
func (c *Client) Exists(srcs ...string) ([]bool, error) {
	res := make([]bool, len(srcs))
	var err error
	for i, src := range srcs {
		if res[i], err = c.ManagerFor(src).Exists(src); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func LocalPath(src string) string {
	f, err := ioutil.TempFile("", filepath.Base(src))
	if err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	return f.Name()
}

// Utility methods.

func detectContentType(name string, r io.ReadSeeker) string {
	first512 := make([]byte, 512)
	r.Read(first512)                             // If it dies... it dies.
	defer r.Seek(0, io.SeekStart)                // Reset our position.
	detected := http.DetectContentType(first512) // Always returns a valid type.
	if detected == "application/octet-stream" {
		// mp4 not detected https: //github.com/golang/go/issues/8773
		if filepath.Ext(name) == ".mp4" {
			return "video/mp4"
		}
	}
	return detected
}

// Parses a bucket name and key name from a cloud url of the format:
// <prefix>//<bucket_name>/<key_name>
func BucketKey(s string) (bucket, key string, err error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", err
	}
	bucket = strings.Trim(u.Host, "/")
	key = strings.Trim(u.Path, "/")
	return bucket, key, nil
}
