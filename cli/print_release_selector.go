package cli

import (
	"fmt"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"strconv"
	"strings"
)

func PrintReleaseSelector(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector) {
	prsa := nod.Begin("GitHub releases selectors:")
	defer prsa.End()

	params := make(map[string][]string)

	for _, os := range operatingSystems {
		params[vangogh_local_data.OperatingSystemsProperty] = append(params[vangogh_local_data.OperatingSystemsProperty], os.String())
	}

	if releaseSelector != nil {
		if releaseSelector.Owner != "" {
			params[data.GitHubOwnerProperty] = append(params[data.GitHubOwnerProperty], releaseSelector.Owner)
		}

		if releaseSelector.Repo != "" {
			params[data.GitHubRepoProperty] = append(params[data.GitHubRepoProperty], releaseSelector.Repo)
		}

		for _, tag := range releaseSelector.Tags {
			params[data.GitHubTagProperty] = append(params[data.GitHubTagProperty], tag)
		}

		if releaseSelector.All {
			params[data.GitHubAllReleasesProperty] = append(params[data.GitHubAllReleasesProperty], strconv.FormatBool(releaseSelector.All))
		}
	}

	pvs := make([]string, 0, len(params))
	for _, p := range []string{
		vangogh_local_data.OperatingSystemsProperty,
		data.GitHubOwnerProperty,
		data.GitHubRepoProperty,
		data.GitHubTagProperty,
		data.GitHubAllReleasesProperty} {

		if _, ok := params[p]; !ok {
			continue
		}

		pvs = append(pvs, fmt.Sprintf("%s=%s", p, strings.Join(params[p], ",")))
	}

	prsa.EndWithResult(strings.Join(pvs, "; "))
}
