package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"strconv"
	"strings"
)

type installParameters struct {
	operatingSystem vangogh_integration.OperatingSystem
	langCode        string
	downloadTypes   []vangogh_integration.DownloadType
	keepDownloads   bool
	noSteamShortcut bool
}

func (ip *installParameters) String() string {

	params := make([]string, 0)

	dts := make([]string, 0, len(ip.downloadTypes))
	for _, dt := range ip.downloadTypes {
		dts = append(dts, dt.String())
	}

	params = append(params, vangogh_integration.OperatingSystemsProperty+"="+ip.operatingSystem.String())
	params = append(params, vangogh_integration.LanguageCodeProperty+"="+ip.langCode)
	params = append(params, vangogh_integration.DownloadTypeProperty+"="+strings.Join(dts, ","))
	params = append(params, data.KeepDownloadsProperty+"="+strconv.FormatBool(ip.keepDownloads))
	params = append(params, data.NoSteamShortcutProperty+"="+strconv.FormatBool(ip.noSteamShortcut))

	return strings.Join(params, ";")
}

func parseInstallParameters(line string) *installParameters {
	ip := new(installParameters)
	for _, parameterValues := range strings.Split(line, ";") {
		if parameter, values, ok := strings.Cut(parameterValues, "="); ok {
			switch parameter {
			case vangogh_integration.OperatingSystemsProperty:
				ip.operatingSystem = vangogh_integration.ParseOperatingSystem(values)
			case vangogh_integration.LanguageCodeProperty:
				ip.langCode = values
			case vangogh_integration.DownloadTypeProperty:
				ip.downloadTypes = vangogh_integration.ParseManyDownloadTypes(strings.Split(values, ","))
			case data.KeepDownloadsProperty:
				ip.keepDownloads = values == "true"
			case data.NoSteamShortcutProperty:
				ip.noSteamShortcut = values == "true"
			}
		}
	}
	return ip
}

func filterInstallParameters(operatingSystem vangogh_integration.OperatingSystem, langCode string, lines ...string) string {
	for _, line := range lines {
		ip := parseInstallParameters(line)
		if ip.operatingSystem == operatingSystem && ip.langCode == langCode {
			return ip.String()
		}
	}
	return ""
}

func defaultInstallParameters(os vangogh_integration.OperatingSystem) *installParameters {
	return &installParameters{
		operatingSystem: os,
		langCode:        defaultLangCode,
		downloadTypes:   []vangogh_integration.DownloadType{vangogh_integration.Installer, vangogh_integration.DLC},
		keepDownloads:   false,
		noSteamShortcut: false,
	}
}

func pinInstallParameters(ip *installParameters, ids ...string) error {

	pipa := nod.Begin(" pinning install parameters...")
	defer pipa.EndWithResult("done")

	printInstallParameters(ip)

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.InstallParametersProperty)
	if err != nil {
		return err
	}

	ips := ip.String()

	installedParams := make(map[string][]string)
	for _, id := range ids {
		installedParams[id] = append(installedParams[id], ips)
	}

	return rdx.BatchAddValues(data.InstallParametersProperty, installedParams)
}

func printInstallParameters(ip *installParameters) {
	pipa := nod.Begin(" install parameters:")
	pipa.EndWithResult(ip.String())
}

func unpinInstallParameters(
	operatingSystem vangogh_integration.OperatingSystem,
	langCode string,
	ids ...string) error {

	uipa := nod.Begin(" unpinning install parameters...")
	defer uipa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.InstallParametersProperty)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if installParams, ok := rdx.GetAllValues(data.InstallParametersProperty, id); ok {
			if olcip := filterInstallParameters(operatingSystem, langCode, installParams...); olcip != "" {
				if err = rdx.CutValues(data.InstallParametersProperty, id, olcip); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
