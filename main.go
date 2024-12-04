package main

import (
	"bytes"
	_ "embed"
	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/clo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
)

var (
	//go:embed "cli-commands.txt"
	cliCommands []byte
	//go:embed "cli-help.txt"
	cliHelp []byte
)

func main() {

	nod.EnableStdOutPresenter()

	ns := nod.Begin("theo is complementing vangogh experience")
	defer ns.End()

	theoRootDir, err := data.InitRootDir()
	if err != nil {
		_ = ns.EndWithError(err)
	}

	if err := pathways.Setup("",
		theoRootDir,
		data.RelToAbsDirs,
		data.AllAbsDirs...); err != nil {
		_ = ns.EndWithError(err)
		os.Exit(1)
	}

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		nil)
	if err != nil {
		_ = ns.EndWithError(err)
		os.Exit(1)
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"backup":                  cli.BackupHandler,
		"cache-github-releases":   cli.CacheGitHubReleasesHandler,
		"cleanup-github-releases": cli.CleanupGitHubReleasesHandler,
		"download":                cli.DownloadHandler,
		"extract":                 cli.ExtractHandler,
		"finalize-installation":   cli.FinalizeInstallationHandler,
		"get-downloads-metadata":  cli.GetDownloadsMetadataHandler,
		"get-github-releases":     cli.GetGitHubReleasesHandler,
		"install":                 cli.InstallHandler,
		"list-installed":          cli.ListInstalledHandler,
		"native-install":          cli.NativeInstallHandler,
		"pin-installed-metadata":  cli.PinInstalledMetadataHandler,
		"place-extracts":          cli.PlaceExtractsHandler,
		"serve":                   cli.ServeHandler,
		"setup":                   cli.SetupHandler,
		"remove-extracts":         cli.RemoveExtractsHandler,
		"remove-downloads":        cli.RemoveDownloadsHandler,
		"reveal-downloads":        cli.RevealDownloadsHandler,
		"reveal-installed":        cli.RevealInstalledHandler,
		"reset-setup":             cli.ResetSetupHandler,
		"test-setup":              cli.TestSetupHandler,
		"uninstall":               cli.UninstallHandler,
		"validate":                cli.ValidateHandler,
		"version":                 cli.VersionHandler,
	})

	if err := defs.AssertCommandsHaveHandlers(); err != nil {
		_ = ns.EndWithError(err)
		os.Exit(1)
	}

	if err := defs.Serve(os.Args[1:]); err != nil {
		_ = ns.EndWithError(err)
		os.Exit(1)
	}
}
