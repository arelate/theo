package pfx_mod

import (
	"bytes"
	_ "embed"
	"github.com/arelate/theo/data"
	"io"
	"os"
	"path/filepath"
)

const (
	retinaOnFilename  = "retina_on.reg"
	retinaOffFilename = "retina_off.reg"
)

var (
	//go:embed "registry/retina_on.reg"
	retinaOnReg []byte
	//go:embed "registry/retina_off.reg"
	retinaOffReg []byte
)

func ToggleRetina(wineCtx *data.WineContext, revert bool, force bool) error {

	absDriveCroot := filepath.Join(wineCtx.PrefixPath, data.RelPfxDriveCDir)

	regFilename := retinaOnFilename
	regContent := retinaOnReg
	if revert {
		regFilename = retinaOffFilename
		regContent = retinaOffReg
	}

	absRegPath := filepath.Join(absDriveCroot, regFilename)
	if _, err := os.Stat(absRegPath); os.IsNotExist(err) || (err == nil && force) {
		if err := createRegFile(absRegPath, regContent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return data.RegeditWinePrefix(wineCtx, absRegPath)
}

func createRegFile(absPath string, content []byte) error {

	regFile, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer regFile.Close()

	if _, err := io.Copy(regFile, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}
