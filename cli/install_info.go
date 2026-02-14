package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const defaultLangCode = "en"

type resolutionPolicy int

const (
	currentOsThenWindows resolutionPolicy = iota
	installedOperatingSystem
	installedLangCode
)

type InstallInfo struct {
	OperatingSystem     vangogh_integration.OperatingSystem `json:"os"`
	LangCode            string                              `json:"lang-code"`
	DownloadTypes       []vangogh_integration.DownloadType  `json:"download-types"`
	DownloadableContent []string                            `json:"dlc"`
	Version             string                              `json:"version"`
	EstimatedBytes      int64                               `json:"estimated-bytes"`
	KeepDownloads       bool                                `json:"keep-downloads"`
	NoSteamShortcut     bool                                `json:"no-steam-shortcut"`
	Env                 []string                            `json:"env"`
	SteamInstall        bool                                `json:"steam-install,omitempty"`
	reveal              bool                                // won't be serialized
	verbose             bool                                // won't be serialized
	force               bool                                // won't be serialized
}

func (ii *InstallInfo) AddProductDetails(pd *vangogh_integration.ProductDetails) {

	dls := pd.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	ii.EstimatedBytes = 0
	for _, dl := range dls {
		if ii.Version == "" && dl.DownloadType == vangogh_integration.Installer {
			ii.Version = dl.Version
		}
		ii.EstimatedBytes += dl.EstimatedBytes
	}
}

func matchInstallInfo(ii *InstallInfo, lines ...string) (*InstallInfo, string, error) {
	for _, line := range lines {
		var installedInfo InstallInfo
		if err := json.UnmarshalRead(strings.NewReader(line), &installedInfo); err != nil {
			return nil, "", err
		}

		if installedInfo.OperatingSystem == ii.OperatingSystem && installedInfo.LangCode == ii.LangCode {
			return &installedInfo, line, nil
		}
	}
	return nil, "", nil
}

func pinInstallInfo(id string, ii *InstallInfo, rdx redux.Writeable) error {

	piia := nod.Begin("pinning install info for %s...", id)
	defer piia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	if exists, err := hasInstallInfo(id, ii, rdx); err == nil && exists {
		if err = unpinInstallInfo(id, ii, rdx); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err := json.MarshalWrite(buf, ii); err != nil {
		return err
	}

	return rdx.BatchAddValues(data.InstallInfoProperty, map[string][]string{id: {buf.String()}})
}

func unpinInstallInfo(id string, ii *InstallInfo, rdx redux.Writeable) error {

	uiia := nod.Begin(" unpinning install info...")
	defer uiia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		_, installedInfoLine, err := matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if err = rdx.CutValues(data.InstallInfoProperty, id, installedInfoLine); err != nil {
			return err
		}

	} else {
		uiia.EndWithResult("install info not found for %s %s-%s", id, ii.OperatingSystem, ii.LangCode)
	}

	return nil
}

func hasInstallInfo(id string, ii *InstallInfo, rdx redux.Readable) (bool, error) {

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return false, err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		installInfo, _, err := matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return false, err
		}

		return installInfo != nil, nil

	} else {
		return false, nil
	}
}

func installedInfoOperatingSystem(id string, rdx redux.Readable) (vangogh_integration.OperatingSystem, error) {

	iiosa := nod.Begin(" checking installed operating system for %s...", id)
	defer iiosa.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return vangogh_integration.AnyOperatingSystem, err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		switch len(installedInfoLines) {
		case 0:
			return vangogh_integration.AnyOperatingSystem, errors.New("zero length installed info for " + id)
		default:

			distinctOs := make([]vangogh_integration.OperatingSystem, 0)

			for _, line := range installedInfoLines {

				var ii InstallInfo
				if err := json.UnmarshalRead(strings.NewReader(line), &ii); err != nil {
					return vangogh_integration.AnyOperatingSystem, err
				}

				if !slices.Contains(distinctOs, ii.OperatingSystem) {
					distinctOs = append(distinctOs, ii.OperatingSystem)
				}

			}

			switch len(distinctOs) {
			case 0:
				return vangogh_integration.AnyOperatingSystem, errors.New("no supported operating system for " + id)
			case 1:
				return distinctOs[0], nil
			default:
				return vangogh_integration.AnyOperatingSystem, errors.New("please specify operating system for " + id)
			}

		}
	} else {
		return vangogh_integration.AnyOperatingSystem, errors.New("no installation found for " + id)
	}
}

func installedInfoLangCode(id string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {
	iilca := nod.Begin(" checking installed language code for %s...", id)
	defer iilca.Done()

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		switch len(installedInfoLines) {
		case 0:
			return "", errors.New("zero length installed info for " + id)
		default:

			distinctLangCodes := make([]string, 0)

			for _, line := range installedInfoLines {

				var ii InstallInfo
				if err := json.UnmarshalRead(strings.NewReader(line), &ii); err != nil {
					return "", err
				}

				if ii.OperatingSystem != operatingSystem {
					continue
				}

				if !slices.Contains(distinctLangCodes, ii.LangCode) {
					distinctLangCodes = append(distinctLangCodes, ii.LangCode)
				}

			}

			switch len(distinctLangCodes) {
			case 0:
				return "", errors.New("no supported language code system for " + id)
			case 1:
				return distinctLangCodes[0], nil
			default:
				return "", errors.New("please specify language code for " + id)
			}

		}
	} else {
		return "", errors.New("no installation found for " + id)
	}
}

func installedInfoSteamInstall(id string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (bool, error) {
	iisia := nod.Begin(" checking if %s is a Steam install...", id)
	defer iisia.Done()

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		switch len(installedInfoLines) {
		case 0:
			return false, errors.New("zero length installed info for " + id)
		default:

			for _, line := range installedInfoLines {

				var ii InstallInfo
				if err := json.UnmarshalRead(strings.NewReader(line), &ii); err != nil {
					return false, err
				}

				if ii.OperatingSystem != operatingSystem {
					continue
				}

				return ii.SteamInstall, nil

			}
		}
	} else {
		return false, errors.New("no installation found for " + id)
	}
	return false, nil
}

func resolveInstallInfo(id string, installInfo *InstallInfo, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable, policies ...resolutionPolicy) error {

	nod.Log("resolveInstallInfo: policies %v", policies)

	if installInfo.OperatingSystem == vangogh_integration.AnyOperatingSystem {

		nod.Log("resolveInstallInfo: resolving %s=%s...",
			vangogh_integration.OperatingSystemsProperty,
			installInfo.OperatingSystem)

		if slices.Contains(policies, currentOsThenWindows) {

			if productDetails == nil {
				return errors.New("product details are required to resolve install info")
			}

			if slices.Contains(productDetails.OperatingSystems, data.CurrentOs()) {
				installInfo.OperatingSystem = data.CurrentOs()
			} else if slices.Contains(productDetails.OperatingSystems, vangogh_integration.Windows) {
				installInfo.OperatingSystem = vangogh_integration.Windows
			} else {
				unsupportedOsMsg := fmt.Sprintf("product doesn't support %s or %s, only %v",
					data.CurrentOs(), vangogh_integration.Windows, productDetails.OperatingSystems)
				return errors.New(unsupportedOsMsg)
			}

		} else if slices.Contains(policies, installedOperatingSystem) {

			installedOs, err := installedInfoOperatingSystem(id, rdx)
			if err != nil {
				return err
			}

			installInfo.OperatingSystem = installedOs
		}

		nod.Log("resolveInstallInfo: resolved %s=%s",
			vangogh_integration.OperatingSystemsProperty,
			installInfo.OperatingSystem)
	}

	if len(installInfo.DownloadTypes) == 0 {

		defaultDownloadTypes := []vangogh_integration.DownloadType{
			vangogh_integration.Installer,
			vangogh_integration.DLC,
		}

		nod.Log("resolveInstallInfo: resolved %s=%v",
			vangogh_integration.DownloadTypeProperty,
			defaultDownloadTypes)

		installInfo.DownloadTypes = defaultDownloadTypes
	}

	if installInfo.LangCode == "" {

		nod.Log("resolveInstallInfo: resolving %s...",
			vangogh_integration.LanguageCodeProperty)

		if slices.Contains(policies, installedLangCode) {

			if lc, err := installedInfoLangCode(id, installInfo.OperatingSystem, rdx); err == nil {
				installInfo.LangCode = lc
			} else {
				return err
			}

		} else {
			installInfo.LangCode = defaultLangCode
		}

		nod.Log("resolveInstallInfo: resolved %s=%s",
			vangogh_integration.LanguageCodeProperty,
			installInfo.LangCode)
	}

	if steamInstall, err := installedInfoSteamInstall(id, installInfo.OperatingSystem, rdx); err == nil {
		installInfo.SteamInstall = steamInstall
	} else {
		return err
	}

	return nil
}

func printInstallInfoParams(ii *InstallInfo, noPatches bool, ids ...string) {
	vangogh_integration.PrintParams(ids,
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		noPatches)
}
