package cli

import "github.com/arelate/southern_light/vangogh_integration"

type execTask struct {
	title              string
	operatingSystem    vangogh_integration.OperatingSystem
	osArch             string
	betaKey            string
	steamLcType        string
	exe                string
	workDir            string
	args               []string
	env                []string
	protonOptions      []string
	protonRuntime      string
	steamProtonRuntime string
	prefix             string
	playTask           string
	noFix              bool
	defaultLauncher    bool
	verbose            bool
}
