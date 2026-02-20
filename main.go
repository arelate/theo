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
		"backup-metadata":  cli.BackupMetadataHandler,
		"connect":          cli.ConnectHandler,
		"download":         cli.DownloadHandler,
		"install":          cli.InstallHandler,
		"list":             cli.ListHandler,
		"prefix":           cli.PrefixHandler,
		"remove-downloads": cli.RemoveDownloadsHandler,
		"reveal":           cli.RevealHandler,
		"run":              cli.RunHandler,
		"setup-steamcmd":   cli.SetupSteamCmdHandler,
		"setup-wine":       cli.SetupWineHandler,
		"steam-fix":        cli.SteamFixHandler,
		"steam-shortcut":   cli.SteamShortcutHandler,
		"uninstall":        cli.UninstallHandler,
		"update":           cli.UpdateHandler,
		"validate":         cli.ValidateHandler,
		"version":          cli.VersionHandler,
	})

	if err = defs.AssertCommandsHaveHandlers(); err != nil {
		log.Fatalln(err)
	}

	var u *url.URL
	if u, err = defs.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	//appInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	//
	//kvAppInfo, err := kevlar.New(appInfoDir, steam_vdf.Ext)
	//if err != nil {
	//	panic(err)
	//}
	//
	//for id := range kvAppInfo.Keys() {
	//
	//	var rcAiKv io.ReadCloser
	//	rcAiKv, err = kvAppInfo.Get(id)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	var aiVdf steam_vdf.ValveDataFile
	//	aiVdf, err = steam_vdf.ReadText(rcAiKv)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	var kv *steam_vdf.KeyValues
	//	kv, err = aiVdf.At(id, "common")
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	fmt.Println(id, kv.Val("name"))
	//
	//	if err = rcAiKv.Close(); err != nil {
	//		panic(err)
	//	}
	//
	//}

	if err = defs.Serve(u); err != nil {
		tsa.Error(err)
		log.Fatalln(err)
	}
}
