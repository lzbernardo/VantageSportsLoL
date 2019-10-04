package files

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type LocalProvider struct {
	tmpDir string
}

func NewLocalProvider(tmpDir string) (*LocalProvider, error) {
	return &LocalProvider{tmpDir: tmpDir}, nil
}

func (l *LocalProvider) Rename(src string, dst string, opts ...FileOption) error {
	return os.Rename(src, dst)
}

func (l *LocalProvider) Read(src string) ([]byte, error) {
	return ioutil.ReadFile(src)
}

func (l *LocalProvider) Copy(src string, dst string, opts ...FileOption) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(dst), 0775)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (l *LocalProvider) Remove(path string) error {
	return os.Remove(path)
}

func (l *LocalProvider) List(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return []string{}, nil
	}
	infos, err := ioutil.ReadDir(path)
	result := make([]string, len(infos))
	for i := range infos {
		result[i] = filepath.Join(path, infos[i].Name())
	}
	return result, nil
}

func (l *LocalProvider) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return err == nil, nil
	}
	return err == nil, err
}
