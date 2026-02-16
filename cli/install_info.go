package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"slices"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const defaultLangCode = "en"

var (
	ErrInstallInfoNotFound = errors.New("install info not found")
	ErrInstallInfoTooMany  = errors.New("multiple installations match request")
)

type InstallInfo struct {
	OperatingSystem     vangogh_integration.OperatingSystem `json:"os"`
	LangCode            string                              `json:"lang-code"`
	Origin              data.Origin                         `json:"origin"`
	DownloadTypes       []vangogh_integration.DownloadType  `json:"download-types"`
	DownloadableContent []string                            `json:"dlc"`
	Version             string                              `json:"version"`
	EstimatedBytes      int64                               `json:"estimated-bytes"`
	KeepDownloads       bool                                `json:"keep-downloads"`
	NoSteamShortcut     bool                                `json:"no-steam-shortcut"`
	Env                 []string                            `json:"env"`
	verbose             bool                                // won't be serialized
	force               bool                                // won't be serialized
}

func (ii *InstallInfo) ReduceProductDetails(pd *vangogh_integration.ProductDetails) {

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

	return rdx.AddValues(data.InstallInfoProperty, id, buf.String())
}

func unpinInstallInfo(id string, ii *InstallInfo, rdx redux.Writeable) error {

	uiia := nod.Begin(" unpinning install info...")
	defer uiia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	installedInfo, err := matchInstalledInfo(id, ii, rdx)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err = json.MarshalWrite(buf, installedInfo); err != nil {
		return err
	}

	return rdx.CutValues(data.InstallInfoProperty, id, buf.String())
}

func hasInstallInfo(id string, request *InstallInfo, rdx redux.Readable) (bool, error) {

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return false, err
	}

	if matchedInstalledInfo, err := matchInstalledInfo(id, request, rdx); err == nil && matchedInstalledInfo != nil {
		return true, nil
	} else if errors.Is(err, ErrInstallInfoTooMany) {
		return true, nil
	} else if errors.Is(err, ErrInstallInfoNotFound) {

	}

	return false, ErrInstallInfoNotFound
}

func matchInstalledInfo(id string, request *InstallInfo, rdx redux.Readable) (*InstallInfo, error) {

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return nil, err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		installedInfo, err := unmarshalInstalledInfo(installedInfoLines...)
		if err != nil {
			return nil, err
		}

		switch len(installedInfo) {
		case 0:
			return nil, ErrInstallInfoNotFound
		case 1:
			return installedInfo[0], nil
		default:
			filteredInstalledInfo := filterInstalledInfo(installedInfo, request)

			switch len(filteredInstalledInfo) {
			case 0:
				return nil, ErrInstallInfoNotFound
			case 1:
				return filteredInstalledInfo[0], nil
			default:
				return nil, ErrInstallInfoTooMany
			}
		}

	} else {
		return nil, ErrInstallInfoNotFound
	}
}

func unmarshalInstalledInfo(lines ...string) ([]*InstallInfo, error) {

	installedInfo := make([]*InstallInfo, 0, len(lines))

	for _, line := range lines {
		var ii InstallInfo

		if err := json.UnmarshalRead(strings.NewReader(line), &ii); err != nil {
			return nil, err
		}

		installedInfo = append(installedInfo, &ii)
	}

	return installedInfo, nil
}

func filterInstalledInfo(installedInfo []*InstallInfo, request *InstallInfo) []*InstallInfo {

	filteredInstalledInfo := make([]*InstallInfo, 0, len(installedInfo))

	for _, ii := range installedInfo {
		if request.OperatingSystem != vangogh_integration.AnyOperatingSystem && ii.OperatingSystem == request.OperatingSystem &&
			request.LangCode != "" && ii.LangCode == request.LangCode &&
			request.Origin != data.UnknownOrigin && ii.Origin == request.Origin {
			filteredInstalledInfo = append(filteredInstalledInfo, ii)
		}
	}

	return filteredInstalledInfo
}

func setInstallInfoDefaults(request *InstallInfo, availableOperatingSystems []vangogh_integration.OperatingSystem) {

	if request.Origin == data.UnknownOrigin {
		request.Origin = data.VangoghGogOrigin
	}

	if request.OperatingSystem == vangogh_integration.AnyOperatingSystem {
		if slices.Contains(availableOperatingSystems, data.CurrentOs()) {
			request.OperatingSystem = data.CurrentOs()
		} else {
			request.OperatingSystem = vangogh_integration.Windows
		}
	}

	if request.LangCode == "" {
		request.LangCode = defaultLangCode
	}

	if len(request.DownloadTypes) == 0 {
		request.DownloadTypes = []vangogh_integration.DownloadType{vangogh_integration.Installer, vangogh_integration.DLC}
	}

}
