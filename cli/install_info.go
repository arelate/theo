package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"strings"
)

type InstallInfo struct {
	OperatingSystem vangogh_integration.OperatingSystem `json:"os"`
	LangCode        string                              `json:"lang-code"`
	DownloadTypes   []vangogh_integration.DownloadType  `json:"download-types"`
	Version         string                              `json:"version"`
	EstimatedBytes  int64                               `json:"estimated-bytes"`
	KeepDownloads   bool                                `json:"keep-downloads"`
	NoSteamShortcut bool                                `json:"no-steam-shortcut"`
	Env             []string                            `json:"env"`
	reveal          bool                                // won't be serialized
	verbose         bool                                // won't be serialized
	force           bool                                // won't be serialized
}

func (ii *InstallInfo) String() (string, error) {

	buf := bytes.NewBuffer(nil)

	if err := json.NewEncoder(buf).Encode(ii); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (ii *InstallInfo) AddProductDetails(pd *vangogh_integration.ProductDetails) {

	for _, dl := range pd.DownloadLinks {
		if dl.OperatingSystem != ii.OperatingSystem {
			continue
		}

		if dl.Type != vangogh_integration.Installer {
			continue
		}

		ii.Version = dl.Version
		ii.EstimatedBytes = dl.EstimatedBytes
		break

	}
}

func parseInstallInfo(line string) (*InstallInfo, error) {
	var ii InstallInfo

	if err := json.NewDecoder(strings.NewReader(line)).Decode(&ii); err != nil {
		return nil, err
	}

	return &ii, nil
}

func matchInstallInfo(ii *InstallInfo, lines ...string) (*InstallInfo, error) {
	for _, line := range lines {
		installedInfo, err := parseInstallInfo(line)
		if err != nil {
			return nil, err
		}
		if installedInfo.OperatingSystem == ii.OperatingSystem && installedInfo.LangCode == ii.LangCode {
			return installedInfo, nil
		}
	}
	return nil, nil
}

func pinInstallInfo(id string, ii *InstallInfo, rdx redux.Writeable) error {

	piia := nod.Begin("pinning install info for %s...", id)
	defer piia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	if err := unpinInstallInfo(id, ii, rdx); err != nil {
		return err
	}

	iis, err := ii.String()
	if err != nil {
		return err
	}

	return rdx.BatchAddValues(data.InstallInfoProperty, map[string][]string{id: {iis}})
}

func unpinInstallInfo(id string, ii *InstallInfo, rdx redux.Writeable) error {

	uiia := nod.Begin(" unpinning install info...")
	defer uiia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		matchedInstalledInfo, err := matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		iis, err := matchedInstalledInfo.String()
		if err != nil {
			return err
		}

		if err = rdx.CutValues(data.InstallInfoProperty, id, iis); err != nil {
			return err
		}

	} else {
		uiia.EndWithResult("install info not found for %s", id)
	}

	return nil
}

func installedInfoOperatingSystem(id string, rdx redux.Readable) (vangogh_integration.OperatingSystem, error) {

	iiosa := nod.Begin(" checking installed operating system for %s...", id)
	defer iiosa.Done()

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		switch len(installedInfoLines) {
		case 0:
			return vangogh_integration.AnyOperatingSystem, errors.New("zero length installed info for " + id)
		case 1:

			ii, err := parseInstallInfo(installedInfoLines[0])
			if err != nil {
				return vangogh_integration.AnyOperatingSystem, err
			}

			iiosa.EndWithResult(ii.OperatingSystem.String())
			return ii.OperatingSystem, nil

		default:
			return vangogh_integration.AnyOperatingSystem, errors.New("please specify OS version for " + id)
		}
	} else {
		return vangogh_integration.AnyOperatingSystem, errors.New("no installation found for " + id)
	}
}
