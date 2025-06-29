package cli

import _ "embed"

const (
	retinaOnFilename  = "retina_on.reg"
	retinaOffFilename = "retina_off.reg"
)

const regeditBin = "regedit"

const (
	prefixModEnableRetina  = "enable-retina"
	prefixModDisableRetina = "disable-retina"
)

var (
	//go:embed "registry/retina_on.reg"
	retinaOnReg []byte
	//go:embed "registry/retina_off.reg"
	retinaOffReg []byte
)

func PrefixMods() []string {
	return []string{
		prefixModEnableRetina,
		prefixModDisableRetina,
	}
}
