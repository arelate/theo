package main

import (
	"bytes"
	_ "embed"
	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/clo_delegates"
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
		log.Fatalln(err.Error())
	}

	if err = pathways.Setup("",
		theoRootDir,
		data.RelToAbsDirs,
		data.AllAbsDirs...); err != nil {
		log.Fatalln(err.Error())
	}

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		clo_delegates.FuncMap)
	if err != nil {
		log.Fatalln(err.Error())
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"backup-metadata":  cli.BackupMetadataHandler,
		"download":         cli.DownloadHandler,
		"install":          cli.InstallHandler,
		"list":             cli.ListHandler,
		"prefix":           cli.PrefixHandler,
		"remove-downloads": cli.RemoveDownloadsHandler,
		"reveal":           cli.RevealHandler,
		"run":              cli.RunHandler,
		"setup-server":     cli.SetupServerHandler,
		"setup-wine":       cli.SetupWineHandler,
		"steam-shortcut":   cli.SteamShortcutHandler,
		"uninstall":        cli.UninstallHandler,
		"update":           cli.UpdateHandler,
		"validate":         cli.ValidateHandler,
		"version":          cli.VersionHandler,
	})

	if err = defs.AssertCommandsHaveHandlers(); err != nil {
		log.Fatalln(err.Error())
	}

	if err = defs.Serve(os.Args[1:]); err != nil {
		log.Fatalln(err.Error())
	}
}
