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
		"add-steam-shortcut":      cli.AddSteamShortcutHandler,
		"archive-prefix":          cli.ArchivePrefixHandler,
		"backup-metadata":         cli.BackupMetadataHandler,
		"default-prefix-env":      cli.DefaultPrefixEnvHandler,
		"delete-prefix-env":       cli.DeletePrefixEnvHandler,
		"delete-prefix-exe-path":  cli.DeletePrefixExePathHandler,
		"download":                cli.DownloadHandler,
		"install":                 cli.InstallHandler,
		"list-installed":          cli.ListInstalledHandler,
		"list-prefix-env":         cli.ListPrefixEnvHandler,
		"list-prefix-exe-path":    cli.ListPrefixExePathHandler,
		"list-steam-shortcuts":    cli.ListSteamShortcutsHandler,
		"list-wine-installed":     cli.ListWineInstalledHandler,
		"migrate":                 cli.MigrateHandler,
		"mod-prefix-retina":       cli.ModPrefixRetinaHandler,
		"serve":                   cli.ServeHandler,
		"set-prefix-env":          cli.SetPrefixEnvHandler,
		"set-prefix-exe-path":     cli.SetPrefixExePathHandler,
		"set-server-connection":   cli.SetServerConnectionHandler,
		"remove-downloads":        cli.RemoveDownloadsHandler,
		"remove-prefix":           cli.RemovePrefixHandler,
		"remove-steam-shortcut":   cli.RemoveSteamShortcutHandler,
		"reveal-backups":          cli.RevealBackupsHandler,
		"reveal-downloads":        cli.RevealDownloadsHandler,
		"reveal-installed":        cli.RevealInstalledHandler,
		"reveal-prefix":           cli.RevealPrefixHandler,
		"reset-server-connection": cli.ResetServerConnectionHandler,
		"run":                     cli.RunHandler,
		"test-server-connection":  cli.TestServerConnectionHandler,
		"uninstall":               cli.UninstallHandler,
		"update-wine":             cli.UpdateWineHandler,
		"validate":                cli.ValidateHandler,
		"version":                 cli.VersionHandler,
		"wine-install":            cli.WineInstallHandler,
		"wine-run":                cli.WineRunHandler,
		"wine-uninstall":          cli.WineUninstallHandler,
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
