package main

import (
	"bytes"
	_ "embed"
	"log"
	"net/url"
	"os"

	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/clo_delegates"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/clo"
	"github.com/boggydigital/nod"
)

var (
	//go:embed "cli-commands.txt"
	cliCommands []byte
	//go:embed "cli-help.txt"
	cliHelp []byte
)

func main() {

	nod.EnableStdOutPresenter()

	tsa := nod.Begin("theo is complementing vangogh experience")
	defer tsa.Done()

	if err := data.InitPathways(); err != nil {
		log.Fatalln(err)
	}

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		clo_delegates.FuncMap)
	if err != nil {
		log.Fatalln(err)
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"backup-metadata":       cli.BackupMetadataHandler,
		"connect":               cli.ConnectHandler,
		"download":              cli.DownloadHandler,
		"fetch-data":            cli.FetchDataHandler,
		"fix":                   cli.FixHandler,
		"install":               cli.InstallHandler,
		"launch-options":        cli.LaunchOptionsHandler,
		"list":                  cli.ListHandler,
		"prefix":                cli.PrefixHandler,
		"preset-launch-options": cli.PresetLaunchOptionsHandler,
		"remove-downloads":      cli.RemoveDownloadsHandler,
		"reveal":                cli.RevealHandler,
		"run":                   cli.RunHandler,
		"setup-steamcmd":        cli.SetupSteamCmdHandler,
		"setup-wine":            cli.SetupWineHandler,
		"steam-shortcut":        cli.SteamShortcutHandler,
		"uninstall":             cli.UninstallHandler,
		"update":                cli.UpdateHandler,
		"validate":              cli.ValidateHandler,
		"version":               cli.VersionHandler,
	})

	if err = defs.AssertCommandsHaveHandlers(); err != nil {
		log.Fatalln(err)
	}

	var u *url.URL
	if u, err = defs.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	if err = defs.Serve(u); err != nil {
		tsa.Error(err)
		log.Fatalln(err)
	}
}
