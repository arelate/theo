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

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		nil)
	if err != nil {
		log.Println(err.Error())
		os.Exit(3)
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"archive-prefix":         cli.ArchivePrefixHandler,
		"backup-metadata":        cli.BackupMetadataHandler,
		"default-prefix-env":     cli.DefaultPrefixEnvHandler,
		"delete-prefix-env":      cli.DeletePrefixEnvHandler,
		"delete-prefix-exe-path": cli.DeletePrefixExePathHandler,
		"download":               cli.DownloadHandler,
		"install":                cli.InstallHandler,
		"list":                   cli.ListHandler,
		"mod-prefix-retina":      cli.ModPrefixRetinaHandler,
		"remove-downloads":       cli.RemoveDownloadsHandler,
		"remove-prefix":          cli.RemovePrefixHandler,
		"reveal":                 cli.RevealHandler,
		"run":                    cli.RunHandler,
		"serve":                  cli.ServeHandler,
		"set-prefix-env":         cli.SetPrefixEnvHandler,
		"set-prefix-exe-path":    cli.SetPrefixExePathHandler,
		"setup-server":           cli.SetupServerHandler,
		"setup-wine":             cli.SetupWineHandler,
		"steam-shortcut":         cli.SteamShortcutHandler,
		"uninstall":              cli.UninstallHandler,
		"update":                 cli.UpdateHandler,
		"validate":               cli.ValidateHandler,
		"version":                cli.VersionHandler,
		"winecfg":                cli.WineCfgHandler,
		"wine-run":               cli.WineRunHandler,
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
