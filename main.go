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

	if err := data.InitGitHubSources(); err != nil {
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
		"archive-prefix":           cli.ArchivePrefixHandler,
		"backup-metadata":          cli.BackupMetadataHandler,
		"cache-github-releases":    cli.CacheGitHubReleasesHandler,
		"check-gstreamer":          cli.CheckGstreamerHandler,
		"cleanup-github-releases":  cli.CleanupGitHubReleasesHandler,
		"download":                 cli.DownloadHandler,
		"get-downloads-metadata":   cli.GetDownloadsMetadataHandler,
		"get-github-releases":      cli.GetGitHubReleasesHandler,
		"init-prefix":              cli.InitPrefixHandler,
		"install":                  cli.InstallHandler,
		"list-installed":           cli.ListInstalledHandler,
		"mod-prefix-dxvk":          cli.ModPrefixDxVkHandler,
		"mod-prefix-retina":        cli.ModPrefixRetinaHandler,
		"pin-installed-metadata":   cli.PinInstalledMetadataHandler,
		"serve":                    cli.ServeHandler,
		"set-vangogh-connection":   cli.SetVangoghConnectionHandler,
		"remove-downloads":         cli.RemoveDownloadsHandler,
		"remove-github-binaries":   cli.RemoveGitHubBinariesHandler,
		"remove-prefix":            cli.RemovePrefixHandler,
		"reveal-backups":           cli.RevealBackupsHandler,
		"reveal-downloads":         cli.RevealDownloadsHandler,
		"reveal-installed":         cli.RevealInstalledHandler,
		"reveal-prefix":            cli.RevealPrefixHandler,
		"reset-vangogh-connection": cli.ResetVangoghConnectionHandler,
		"test-vangogh-connection":  cli.TestVangoghConnectionHandler,
		"uninstall":                cli.UninstallHandler,
		"unpack-github-releases":   cli.UnpackGitHubReleasesHandler,
		"update-prefix":            cli.UpdatePrefixHandler,
		"update-wine":              cli.UpdateWineHandler,
		"validate":                 cli.ValidateHandler,
		"version":                  cli.VersionHandler,
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
