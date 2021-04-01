package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

type UploadWorkspace struct {
	WsType WorkspaceType

	ConfigFilepath string

	Manifest *Manifest

	base string
	done bool
}

func (uw *UploadWorkspace) Dir() string {
	return uw.base
}

func (uw *UploadWorkspace) Done() bool {
	return uw.done
}

func (uw *UploadWorkspace) DumpTemp() error {
	path := filepath.Join(uw.base, ManifestFilename) + ".tmp"
	return uw.Manifest.Dump(path)
}

func (uw *UploadWorkspace) Finalize() error {
	return os.Rename(filepath.Join(uw.base, ManifestFilename)+".tmp", filepath.Join(uw.base, ManifestFilename))
}

func (uw *UploadWorkspace) FileCount() int {
	return len(uw.Manifest.DataFiles)
}

func (uw *UploadWorkspace) AbsPath(df DataFile) string {
	return filepath.Join(uw.base, df.Filename)
}

func (uw *UploadWorkspace) ForEach(cb func(DataFile) error) error {
	for _, f := range uw.Manifest.DataFiles {
		err := cb(*f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uw *UploadWorkspace) SetOnchainID(filename, newId string) error {
	if uw.done {
		return xerrors.New("workspace already finailized")
	}
	found := false
	for _, f := range uw.Manifest.DataFiles {
		if f.Filename == filename {
			if f.OnchainID != "" {
				return xerrors.Errorf("old onchain ID %s for %s, new onchain ID %s", f.OnchainID, filename, newId)
			}
			f.OnchainID = newId
			found = true
			break
		}
	}
	if !found {
		return xerrors.Errorf("SetOnchainID failed: %s not found, onchain ID %s", filename, newId)
	}
	return uw.DumpTemp()
}

func LoadUploadWorkspace(dirname string) (Workspace, error) {

	expand, err := homedir.Expand(dirname)
	if err != nil {
		return nil, xerrors.Errorf("failed to expand %s: %w", dirname, err)
	}
	expand, err = filepath.Abs(expand)
	if err != nil {
		return nil, xerrors.Errorf("failed to get absolute path of %s: %w", dirname, err)
	}

	ws := UploadWorkspace{
		WsType: WorkspaceUpload,
		base:   expand,
	}

	err = filepath.Walk(expand, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if expand == path {
			//ignore current folder
			return nil
		}

		switch {
		case strings.HasPrefix(f.Name(), "config"):
			if ws.ConfigFilepath != "" {
				return xerrors.New("duplicate 'config*' files found")
			}
			ws.ConfigFilepath = path
		case f.Name() == ManifestFilename:
			ws.done = true
			fallthrough
		case f.Name() == ManifestFilename+".tmp":
			ws.Manifest, err = loadManifest(path)
			if err != nil {
				return err
			}
		default:
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ws.ConfigFilepath == "" {
		return nil, xerrors.New("'config*' file not found")
	}
	if ws.Manifest == nil {
		ws.Manifest, err = newManifest(ws.ConfigFilepath)
		if err != nil {
			return nil, err
		}
		if err = ws.DumpTemp(); err != nil {
			return nil, err
		}
	}
	return &ws, nil
}
