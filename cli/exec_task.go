package cli

type execTask struct {
	name               string
	exe                string
	workDir            string
	args               []string
	env                []string
	protonOptions      []string
	protonRuntime      string
	steamProtonRuntime string
	prefix             string
	playTask           string
	defaultLauncher    bool
	verbose            bool
}
