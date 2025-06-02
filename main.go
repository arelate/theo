package main

import (
	"bytes"
	_ "embed"
	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/clo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"log"
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
	defer ns.Done()

	theoRootDir, err := data.InitRootDir()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	if err = pathways.Setup("",
		theoRootDir,
		data.RelToAbsDirs,
		data.AllAbsDirs...); err != nil {
		log.Println(err.Error())
		os.Exit(2)
	}

	valueDelegates := map[string]func() []string{
		"steam-shortcut-targets": cli.AllSteamShortcutTargets,
	}

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		valueDelegates)
	if err != nil {
		log.Println(err.Error())
		os.Exit(3)
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"add-steam-shortcut":      cli.AddSteamShortcutHandler,
		"archive-prefix":          cli.ArchivePrefixHandler,
		"backup-metadata":         cli.BackupMetadataHandler,
		"default-prefix-env":      cli.DefaultPrefixEnvHandler,
		"delete-prefix-env":       cli.DeletePrefixEnvHandler,
		"delete-prefix-exe-path":  cli.DeletePrefixExePathHandler,
		"download":                cli.DownloadHandler,
		"get-product-details":     cli.GetProductDetailsHandler,
		"has-free-space":          cli.HasFreeSpaceHandler,
		"install":                 cli.InstallHandler,
		"list-installed":          cli.ListInstalledHandler,
		"list-playtasks":          cli.ListPlayTasksHandler,
		"list-prefix-env":         cli.ListPrefixEnvHandler,
		"list-prefix-exe-path":    cli.ListPrefixExePathHandler,
		"list-steam-shortcuts":    cli.ListSteamShortcutsHandler,
		"list-wine-installed":     cli.ListWineInstalledHandler,
		"mod-prefix-retina":       cli.ModPrefixRetinaHandler,
		"serve":                   cli.ServeHandler,
		"set-prefix-env":          cli.SetPrefixEnvHandler,
		"set-prefix-exe-path":     cli.SetPrefixExePathHandler,
		"set-server-connection":   cli.SetServerConnectionHandler,
		"setup-wine":              cli.SetupWineHandler,
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
		"update":                  cli.UpdateHandler,
		"validate":                cli.ValidateHandler,
		"version":                 cli.VersionHandler,
		"winecfg":                 cli.WineCfgHandler,
		"wine-install":            cli.WineInstallHandler,
		"wine-run":                cli.WineRunHandler,
		"wine-uninstall":          cli.WineUninstallHandler,
		"wine-update":             cli.WineUpdateHandler,
	})

	if err = defs.AssertCommandsHaveHandlers(); err != nil {
		log.Println(err.Error())
		os.Exit(4)
	}

	if err = defs.Serve(os.Args[1:]); err != nil {
		log.Println(err.Error())
		os.Exit(5)
	}
}
