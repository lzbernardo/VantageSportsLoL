package lolobserver

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/VantageSports/common/files"
)

// FileSaver is a file-persistence abstraction provided to the replay downloader
// that handles saving files to a remote directory.
type FileSaver interface {
	SaveAs(v interface{}, name string, isJSON bool, maxAttempts int) error
}

type filesClientSaver struct {
	fc       *files.Client
	destDir  string
	localDir string
}

func NewFileSaver(fc *files.Client, remoteDir string) (*filesClientSaver, error) {
	localDir, err := ioutil.TempDir("", "file_saver")
	if err != nil {
		return nil, err
	}

	return &filesClientSaver{
		fc:       fc,
		localDir: localDir,
		destDir:  remoteDir,
	}, nil
}

func (fs *filesClientSaver) SaveAs(v interface{}, name string, isJSON bool, maxAttempts int) error {
	localpath := getPath(fs.localDir, name)
	bytes, err := getBytes(v, isJSON)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(localpath, bytes, 0664); err != nil {
		return err
	}

	destpath := getPath(fs.destDir, name)
	attrs := []files.FileOption{}
	if isJSON {
		attrs = append(attrs, files.ContentType("application/json"))
	}
	for i := 0; i < maxAttempts; i++ {
		err = fs.fc.Move(localpath, destpath, attrs...)
		if err == nil {
			return nil
		}
	}
	return err
}

func (fs *filesClientSaver) Close() {
	os.RemoveAll(fs.localDir)
}

func getPath(dir, base string) string {
	return strings.TrimSuffix(dir, "/") + "/" + strings.TrimPrefix(base, "/")
}

func getBytes(v interface{}, isJSON bool) ([]byte, error) {
	if isJSON {
		return json.Marshal(v)
	}
	return v.([]byte), nil
}
