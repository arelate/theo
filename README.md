# theo

`theo` is a command-line (CLI) client for [vangogh](https://github.com/arelate/vangogh) that allows downloading, installing and running games from your local DRM-free collection. At the moment `theo` implements support for macOS and Linux, tested on Steam Deck. Eventually `theo` hopes to provide a web GUI to manage your local games (see [longer term vision](#Longer_term_theo_vision))

## Installing theo

To install `theo` you'll need Go installed on your machine. Please follow [official instructions](https://go.dev/doc/install) to do that. You might want to verify that Go binaries folder is part of your PATH or you'll need to navigate to `~/usr/go/bin` to run it.

When Go is properly installed, you can then run `go install github.com/arelate/theo@latest` to install the latest available version of `theo`.

### Installing on Steam Deck

At the moment `theo` doesn't provide binary distribution or other convenient ways to be installed on a Steam Deck. It can be used on a Steam Deck with the following steps:

- Use any computer with Go installed - doesn't matter what operating system, all you need is a recent Go version
- Clone the project and navigate to the source code folder
- Compile for Steam Deck: `GOOS=linux GOARCH=amd64 go build -o theo`
- Copy `theo` to a Steam Deck. Doesn't really matter where as `theo` will detect it's own location and will function correctly. For convenience you might consider putting the binary in the `theo` state folder (~/.local/share/theo)
- Navigate to that location with a terminal application of your choice and make `theo` executable `chmod +x theo`
- (You are ready to use `theo`!)
- Run `theo` commands with `./theo` or add that location to your PATH

### Longer term theo vision


In the future `theo` hopes to provide a web GUI to perform all supported operations. Current idea is that this will be based (and share code with) on `vangogh` with additions related to `theo` operations.

At the moment `theo` development efforts are focused on achieving reliability, functional correctness and robustness of all operations.

## Configure theo to connect to vangogh

Before using any of the commands below, `theo` needs to be set up to connect to `vangogh`:

`theo set-server-connection -protocol <protocol> -address <address> -port <port> -username <username> -password <password>` - you need to provide address (e.g. vangogh.example.com), username and password for users authorized to download from that `vangogh` instance (this is set in `vangogh` configuration). Other parameters are optional.

`theo test-server-connection` - will perform a series of connectivity and authorization tests to validate your settings. You should see `healthy` if the test completes successfully.

This is only needed to be done once and `theo` will use those settings from that moment and should work until `vangogh` configuration has changed (when different username/password are set). If you ever need to reset this configuration, `reset-server-connection` can help with that.

## Installing native versions with theo

Basic usage of `theo` comes down to the following commands:

`theo install <gog-game-id>` - will install current OS version of a game

`theo run <gog-game-id>` - will run installed version of a game

`theo uninstall <gog-game-id> -force` - will uninstall a game

More helpful commands:

`theo list-installed` - will print all currently installed games

`theo reveal-installed <gog-game-id>` - will open the directory containing game installation

## Installing Windows versions on macOS, Linux with theo

`theo` provides a set of commands to manage Windows versions on macOS, Linux using [WINE](http://winehq.org):

`theo wine-install <gog-game-id>` - will install Windows version of a game

`theo wine-run <gog-game-id>` - will run installed version of a game

`theo wine-uninstall <gog-game-id> -force` - will uninstall a game

On macOS this functionality **requires** a version of [CrossOver](https://www.codeweavers.com/crossover) purchased and licensed for the current user. `theo` assumes CrossOver bundle is located in `/Applications`. Please note that `theo` uses CrossOver 25 environment variables that won't work in version 24 or earlier. At the moment I'm planning to maintain support for the latest version of CrossOver only. In the future alternatives will be considered (e.g. Kegworks, binary WINE distributions, etc).

On Linux `theo` will use [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) and [umu-proton](https://github.com/Open-Wine-Components/umu-proton). You don't need to do anything, theo will download the latest versions for you automatically, as needed. Those will be installed under `theo` state folder, not globally.

More helpful commands:

`theo list-wine-installed` - will print all currently installed Windows versions

## Advanced usage scenarios

### Using specific language version

theo allows you to specify [language](https://en.wikipedia.org/wiki/List_of_ISO_639_language_codes) version to use, when available on GOG and setup on vangogh:

`theo install <gog-game-id> -lang-code ja` - will install Japanese version of a game (if it's available)

`theo run <gog-game-id> -lang-code ja` - will run Japanese version of a game (if it's installed)

You need to specify language code for every command (e.g. `run`, `uninstall`, `reveal-installed`). With this functionality you can have multiple version of the same game in different languages installed at the same time (applies to native and Windows versions).

### Environment variables for Windows versions on another operating systems

theo helps you manage Windows versions installations and [WINE](http://winehq.org) options with environmental variables:
    
`theo set-prefix-env <gog-game-id> -env VAR1=VAL1 VAR2=VAL2` - will set VAR1 and VAR2 environmental variables for this game installation and those values (VAL1 and VAL2) will be used for any commands under those installations (e.g. `wine-run`)

More helpful commands:

`theo default-prefix-env <gog-game-id>` - will reset all environment variables to default values for the current operating system

`theo delete-prefix-env <gog-game-id>` - will completely removed all environment variables (even default ones) for this game installation

`theo list-prefix-env` - will print all environment variables for every game installed with theo

NOTE: Default environment variables set for each operating systems are listed here: [cli/default_prefix_env.go](https://github.com/arelate/theo/blob/main/cli/default_prefix_env.go#L12).

### Steam shortcuts

theo will automatically add Steam shortcuts (when Steam installation is detected) and will include artwork provided by vangogh. Steam shortcuts are automatically added during `install` and removed during `uninstall` (use `-no-steam-shortcut` if that's not desired).

While this functionality has been designed around Steam Deck, it works equally well for Steam Big Picture mode on a desktop operating system.

NOTE: Steam shortcuts are added for all users currently logged into Steam under current operating system user account.

Additionally you can use the following commands to manage Steam shortcuts:

`theo add-steam-shortcut <gog-game-id>` - will add Steam shortcut for a game

`theo remove-steam-shortcut <gog-game-id>` - will remove Steam shortcut for a game

`theo list-steam-shortcuts` - will list all existing Steam shortcuts

### Retina mode for Windows versions on macOS

theo provides `mod-prefix-retina` command to set Retina mode. See more information [here](https://gitlab.winehq.org/wine/wine/-/wikis/Commands/winecfg#screen-resolution-dpi-setting). To revert this change use the command with a `revert` flag, e.g.:
    
`theo mod-prefix-retina <gog-game-id> -revert`

### Where is theo data stored?

theo stores all state under the current user data directory:
- on Linux that is ~/.local/share/theo
- on macOS that is ~/Library/Application Support/theo

## vangogh technical decisions and resulting theo behaviors

`vangogh` is build around storing installers (programs that contain all game data and perform copying of game data and related dependencies on a local machine) from GOG. This is a deliberate decision as by itself `vangogh` can operate completely independently without any client. You can download an installer from `vangogh`, run it and have the game installed. In some cases this becomes quite unpractical - consider Cyberpunk 2077 that is 28 installer files and 11 more installer files for the Phantom Liberty DLC.

As a result - `theo` was built around this to provide convenience. Installing Cyberpunk with DLC will take a single command (or eventually a single click/tap). The fact that `vangogh` uses installers however implies certain behaviours that would be worth calling out:

- `theo` doesn't implement support for every compression or packaging format and relies on installers doing their jobs. As a result there's no good way to report progress of the installation. `theo` can report download progress, validation progress, etc - but running an installer is an atomic operation. 
- installers need to be downloaded and present to work, which means that disk space requirements are doubled. You need space for the installer and installation. Installers are removed upon successful installation, but must be present while installation is in progress. This applies to both native and WINE installations. Using Cyberpunk 2077 as an example you'll need >300Gb for the installation (= 110Gb for the game itself, 40Gb for the DLC and at least the same amount for the resulting installation)
- updates are achieved by installing a new version on top of the old one (not as a partial updated data), which makes the disk space problem even worse (though keep in mind that the installer will overwrite existing files, so it'll still be double requirements, not much worse)

### Can that be solved better?

There are few alternatives to consider to solve this better, that might be implemented later:

1. use application depots that GOG Galaxy uses and transition `vangogh` to use those. Considerations:
   - Pros: this would solve both progress and double disk space requirements. This would also be aligned with Steam depots and solving this would allow vangogh to support DRM-free Steam games (and potentially more stores, e.g. Epic)
   - Cons: this would defeat independent nature of installers. Potentially this con can be solved by providing server-side ability to package depots into installers on demand. This needs to be investigated further: how to package files into installers, how to handle dependencies, how to handle installer scripts, etc - potentially the way GOG Galaxy handles depots likely provides an answer to all those questions.
2. unpack installers on the `vangogh` server on demand, adding a step in the installation process. Considerations:
   - Pros: this would allow to continue relying on installers as the primary stored artifacts
   - Cons: this adds additional storage requirements on `vangogh` server, especially if unpacked installers are not discarded right away and are cached. This doesn't solve patching unless the previous version is also available (which by default is actually true today). In that case both versions would need to be unpacked and diffed to create a patch. In general - this doesn't solve the underlying issue of unpacking installers and just pushes it to the server. While certain things might be less of a problem, e.g. disk space - others will continue to be problematic, e.g. very long time to unpack large games with no good way to report progress.
3. specifically for the patches - GOG provides them in certain cases, however the way they're implemented is not great and unless something changes radically that won't be a great solution:
   - Pros: updates might be easier in terms of disk space requirements 
   - Cons: patches are not used in 100% of situations on GOG. Sometimes there's a patch. Sometimes there's an installer update. GOG also sometimes provides several patches from different versions to another version. Patches metadata is not very formal either - it's all part of the name, which would require heuristics to find the right patch for a given version to another version as well as determining which version is the latest. And of course that would only solve updates problem, but not the others. 

At the moment the first option seems the most beneficial and might get investigated in the future. One of the two must happen to allow more confidence in that path:
- depots are implemented and validated as a viable option solving known limitations and preserving `vangogh` independence from `theo` (e.g. installers or archives are still available as an option somehow - either on demand or as optionally stored)
- `vangogh` independence becomes lower priority than resolving those limitations and newer features (e.g. Steam DRM-free games) combined. In this case `theo` will be required to download, install games from `vangogh`.

## Why is it called theo?

Theodorus van Gogh (1 May 1857 – 25 January 1891) was a Dutch art dealer and the younger brother of Vincent van Gogh. Known as Theo, his support of his older brother's artistic ambitions and well-being allowed Vincent to devote himself entirely to painting. As an art dealer, Theo van Gogh played a crucial role in introducing contemporary French art to the public.

## Windows client?

At the moment `theo` is focused on macOS and Linux support. Implementing (native installations) Windows support in the future should be relatively straightforward - a good starting point would be adding actual implementations for stub functions in [cli/windows_support.go](https://github.com/arelate/theo/blob/main/cli/windows_support.go) and then testing assumptions - e.g. [data/user_dirs.go](https://github.com/arelate/theo/blob/main/data/user_dirs.go). 

At the moment this is not a priority, but might happen in the future.