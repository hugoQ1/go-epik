package workspace

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/xerrors"
)

type WorkspaceType int

const (
	WorkspaceUpload WorkspaceType = iota
	WorkspaceDownload
)

const (
	UploadTaskFilename   = "upload.tmp"
	DownloadTaskFilename = "download.tmp"

	ManifestFilename = "manifest"
)

type Workspace interface {
	Dir() string
	Done() bool
	DumpTemp() error
	Finalize() error
	FileCount() int
	AbsPath(DataFile) string
	ForEach(func(DataFile) error) error
	SetOnchainID(abspath, id string) error
}

type DataFile struct {
	Filename  string
	OnchainID string // piece CID
	Size      uint64
}

type Manifest struct {
	RawFilename string
	RawContent  string //hex-encoded

	DataFiles []*DataFile
}

func (m *Manifest) Dump(path string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, data, 0744); err != nil {
		return xerrors.Errorf("failed to dump manifest %s: %w", path, err)
	}
	return nil
}

func loadManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)

	var mf Manifest
	err = json.Unmarshal(data, &mf)
	if err != nil {
		return nil, xerrors.Errorf("failed to load manifest: %w", err)
	}
	return &mf, nil
}

func newManifest(cfgPath string) (*Manifest, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, xerrors.New("failed to read upload config")
	}

	mf := &Manifest{
		RawFilename: filepath.Base(cfgPath),
		RawContent:  hex.EncodeToString(src),
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	// skip 2 lines
	scanner.Scan()
	scanner.Scan()

	parent := filepath.Dir(cfgPath)
	for scanner.Scan() {
		arr := strings.Fields(scanner.Text())
		if len(arr) == 4 {
			absPath := filepath.Join(parent, arr[2])
			exist, err := FileExists(absPath)
			if err != nil {
				return nil, err
			}
			if !exist {
				return nil, xerrors.Errorf("file not found: %s", arr[2])
			}
			size, err := strconv.Atoi(arr[3])
			if err != nil {
				return nil, err
			}
			mf.DataFiles = append(mf.DataFiles, &DataFile{
				Filename: arr[2],
				Size:     uint64(size),
			})
		}
	}
	return mf, nil
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
