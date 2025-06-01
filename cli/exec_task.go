package cli

type execTask struct {
	exe     string
	workDir string
	args    []string
	env     []string
	verbose bool
}
