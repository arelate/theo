package pfx_mod

import (
	"fmt"
	"path/filepath"
)

const (
	dxVkDlls64Glob = "x64/*.dll"
	dxVkDlls32Glob = "x32/*.dll"
)

const (
	pfxSystem64Path = "windows/system32"
	pfxSystem32Path = "windows/syswow64"
)

func ToggleDxVk(absPrefixDir, absAssetDir string, flag bool, force bool) error {

	// identify x32/x64 release binaries for DXVK-macos (abs paths)
	// locate target abs path in the prefix
	// copy originals to allow restoration
	// based on the flag copy over / restore .dll files

	x64pattern := filepath.Join(absAssetDir, "*", dxVkDlls64Glob)
	matches, err := filepath.Glob(x64pattern)
	if err != nil {
		return err
	}

	//absTargetDir := filepath.Join(absPrefixDir, data.RelPfxDriveCDir,)

	fmt.Println(matches)

	return nil
}
