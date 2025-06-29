package cli

func WinePrograms() []string {
	// https://gitlab.winehq.org/wine/wine/-/wikis/Commands/
	return []string{
		"cacls",           // edit ACLs
		"clock",           // displays a rudimentary clock
		"cmd",             //Command prompt implementation
		"control",         // Control Panel implementation
		"eject",           // ejects optical discs (note that wine eject is different from the normal eject command)
		"expand",          // expands cabinet (.cab) files
		"explorer",        // explorer.exe implementation
		"hh",              // HTML help (.chm file) viewer
		"icinfo",          // shows installed video compressors for Wine
		"iexplore",        // Internet Explorer implementation
		"lodctr",          // load performance monitor counters
		"msiexec",         // msiexec.exe implementation for installing .msi files
		"net",             // starts and stops services
		"notepad",         // Notepad, a simple text editor
		"oleview",         // allows browsing and exploring COM objects as well as configuring DCOM
		"progman",         // Program Manager implementation
		"reg",             // console-based registry editor
		"regedit",         // Registry Editor implementation
		"regsvr32",        // register OLE components in the registry
		"rpcss",           // quasi-implementation of rpcss.exe
		"rundll32",        // loads a DLL and runs an entry point with the specified parameters
		"secedit",         // security configuration edit command
		"services",        // manages services
		"spoolsv",         // print wrapper
		"start",           // starts a program or opens a document in the program normally used for files with that suffix
		"svchost",         // (internal) host process for services
		"taskmgr",         // Task Manager implementation
		"uninstaller",     // basic program uninstaller
		"unlodctr",        // unload performance monitor counters
		"view",            // metafile viewer
		"wineboot",        // "reboots" (restarts) Wine, for when Windows apps call for a real reboot
		"winebrowser",     // launches native OS browser or mail client
		"winecfg",         // GUI configuration tool for Wine
		"wineconsole",     // displays Windows console
		"winedbg",         // Wine debugger core
		"winedevice",      // (internal) manages devices
		"winefile",        // file explorer implementation
		"winemenubuilder", // helps to build Unix menu entries
		"winemine",        // classic minesweeper game
		"winepath",        // translates between Windows and Unix paths formats
		"winetest",        // all the DLL conformance test programs suitable for unattended testing and report submission
		"winevdm",         // Wine virtual DOS program
		"winhelp",         // Help viewer
		"winhlp32",        // Help viewer (32-bit)
		"winver",          // displays an "about Wine" window
		"wordpad",         // wordpad.exe implementation
		"write",           // starts wordpad (for Win16 compatibility)
		"xcopy",           // Wine-compatible xcopy program
	}
}
