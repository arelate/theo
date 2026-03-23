package cli

type execTask struct {
	title              string
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
