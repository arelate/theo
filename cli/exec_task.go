package cli

type execTask struct {
	name            string
	exe             string
	workDir         string
	args            []string
	env             []string
	prefix          string
	playTask        string
	defaultLauncher bool
	verbose         bool
}
