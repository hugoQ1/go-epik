package workspace

import (
	"encoding/hex"
	"io/ioutil"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

type DownloadWorkspace struct {
	WsType WorkspaceType

	ConfigFilepath string

	Manifest *Manifest

	base string
	done bool
}

func (dw *DownloadWorkspace) Dir() string {
	return dw.base
}

func (dw *DownloadWorkspace) Done() bool {
	exist, err := FileExists(filepath.Join(dw.base, dw.Manifest.RawFilename))
	if err != nil {
		panic(err)
	}
	return exist
}

func (dw *DownloadWorkspace) DumpTemp() error {
	return xerrors.New("DumpTemp unsupported")
}

func (dw *DownloadWorkspace) Finalize() error {
	data, err := hex.DecodeString(dw.Manifest.RawContent)
	if err != nil {
		return err
	}
	path := filepath.Join(dw.base, dw.Manifest.RawFilename)
	if err = ioutil.WriteFile(path, data, 0744); err != nil {
		return xerrors.Errorf("finalize failed: %s, %w", path, err)
	}
	return nil
}

func (dw *DownloadWorkspace) FileCount() int {
	return len(dw.Manifest.DataFiles)
}

func (dw *DownloadWorkspace) AbsPath(df DataFile) string {
	return filepath.Join(dw.base, df.Filename)
}

func (dw *DownloadWorkspace) ForEach(cb func(DataFile) error) error {
	for _, f := range dw.Manifest.DataFiles {
		err := cb(*f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dw *DownloadWorkspace) SetOnchainID(filename, newId string) error {
	return xerrors.New("SetOnchainID unsupported")
}

func LoadDownloadWorkspace(dirname string) (Workspace, error) {

	expand, err := homedir.Expand(dirname)
	if err != nil {
		return nil, xerrors.Errorf("failed to expand %s: %w", dirname, err)
	}
	expand, err = filepath.Abs(expand)
	if err != nil {
		return nil, xerrors.Errorf("failed to get absolute path of %s: %w", dirname, err)
	}

	ws := DownloadWorkspace{
		WsType: WorkspaceDownload,
		base:   expand,
	}

	ws.Manifest, err = loadManifest(filepath.Join(expand, ManifestFilename))
	if err != nil {
		return nil, xerrors.Errorf("failed to load manifest: %w", err)
	}

	return &ws, nil
}
