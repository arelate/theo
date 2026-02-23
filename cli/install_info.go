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

const (
	langCodeAny     = ""
	langCodeDefault = "en"
)

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
	TimeUpdated         string                              `json:"time-updated,omitempty"`
	EstimatedBytes      int64                               `json:"estimated-bytes"`
	KeepDownloads       bool                                `json:"keep-downloads"`
	NoSteamShortcut     bool                                `json:"no-steam-shortcut"`
	Env                 []string                            `json:"env"`
	verbose             bool                                // won't be serialized
	force               bool                                // won't be serialized
}

func (ii *InstallInfo) applyOriginData(id string, originData *data.OriginData) error {

	switch ii.Origin {
	case data.VangoghGogOrigin:

		if originData.ProductDetails == nil {
			return errors.New("cannot sync nil product details for " + id)
		}

		originData.OperatingSystems = originData.ProductDetails.OperatingSystems

		dls := originData.ProductDetails.DownloadLinks.
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

		switch originData.ProductDetails.ProductType {
		case vangogh_integration.DlcProductType:
			return fmt.Errorf("install %s required product(s) to get this downloadable content", strings.Join(originData.ProductDetails.RequiresGames, ","))
		case vangogh_integration.PackProductType:
			return fmt.Errorf("install %s included product(s) to get this edition", strings.Join(originData.ProductDetails.IncludesGames, ","))
		case vangogh_integration.GameProductType:
			// do nothing
		default:
			return errors.New("unknown product type " + originData.ProductDetails.ProductType)
		}

	case data.SteamOrigin:

		if originData.AppInfoKv == nil {
			return errors.New("cannot sync nil appinfo for " + id)
		}

		var osList string
		if ol, ok := originData.AppInfoKv.Val(id, "common", "oslist"); ok {
			osList = ol
		}

		originData.OperatingSystems = vangogh_integration.ParseManyOperatingSystems(strings.Split(osList, ","))

		if buildId, ok := originData.AppInfoKv.Val(id, "depots", "branches", "public", "buildid"); ok && buildId != "" {
			ii.Version = buildId
		}

		if timeUpdated, ok := originData.AppInfoKv.Val(id, "depots", "branches", "public", "timeupdated"); ok && timeUpdated != "" {
			ii.TimeUpdated = timeUpdated
		}

		// TODO: implement size reduction for AppInfo

	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}

	return nil
}

func (ii *InstallInfo) Matches(another *InstallInfo) bool {

	var matchesOs, matchesLangCode, matchesOrigin bool

	if ii.OperatingSystem == another.OperatingSystem ||
		ii.OperatingSystem == vangogh_integration.AnyOperatingSystem ||
		another.OperatingSystem == vangogh_integration.AnyOperatingSystem {
		matchesOs = true
	}

	if ii.LangCode == another.LangCode ||
		ii.LangCode == langCodeAny ||
		another.LangCode == langCodeAny {
		matchesLangCode = true
	}

	if ii.Origin == another.Origin ||
		ii.Origin == data.UnknownOrigin ||
		another.Origin == data.UnknownOrigin {
		matchesOrigin = true
	}

	return matchesOs && matchesLangCode && matchesOrigin
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

func unpinInstallInfo(id string, request *InstallInfo, rdx redux.Writeable) error {

	uiia := nod.Begin(" unpinning install info...")
	defer uiia.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return err
	}

	installedInfo, err := matchInstalledInfo(id, request, rdx)
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
		return false, nil
	}

	return false, nil
}

func matchInstalledInfo(id string, request *InstallInfo, rdx redux.Readable) (*InstallInfo, error) {

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return nil, err
	}

	var installedInfo []InstallInfo

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		var err error
		if installedInfo, err = unmarshalInstalledInfo(installedInfoLines...); err != nil {
			return nil, err
		}

	} else {
		return nil, ErrInstallInfoNotFound
	}

	var matchedInstalledInfo []InstallInfo

	for _, ii := range installedInfo {
		if ii.Matches(request) {
			matchedInstalledInfo = append(matchedInstalledInfo, ii)
		}
	}

	switch len(matchedInstalledInfo) {
	case 0:
		return nil, ErrInstallInfoNotFound
	case 1:
		return &matchedInstalledInfo[0], nil
	default:
		return nil, ErrInstallInfoTooMany
	}

}

func unmarshalInstalledInfo(lines ...string) ([]InstallInfo, error) {

	installedInfo := make([]InstallInfo, 0, len(lines))

	for _, line := range lines {
		var ii InstallInfo

		if err := json.UnmarshalRead(strings.NewReader(line), &ii); err != nil {
			return nil, err
		}

		installedInfo = append(installedInfo, ii)
	}

	return installedInfo, nil
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
		request.LangCode = langCodeDefault
	}

	if len(request.DownloadTypes) == 0 {
		request.DownloadTypes = []vangogh_integration.DownloadType{vangogh_integration.Installer, vangogh_integration.DLC}
	}

}
