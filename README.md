# theo

theo is a command-line (CLI) client for [vangogh](https://github.com/arelate/vangogh) that allows downloading, installing, etc games from your local DRM-free collection. At the moment theo implements support for macOS and Linux (tested on Steam Deck).

## Installing theo

To install theo you'll need a Go installed on your machine. Please follow [official instructions](https://go.dev/doc/install) to do that.

Assuming Go is properly installed, you can then run `go install github.com/arelate/theo@latest` to install the latest available version of theo.

## Configure theo to connect to vangogh

Before using any of the commands below, theo needs to be set up to connect to vangogh:

    - `theo set-vangogh-connection <protocol> <address> <port> <username> <password>` - you need to provide at the very least address (e.g. vangogh.example.com), username and password for users authorized to download from that vangogh instance
    - `theo test-vangogh-connection` - will perform a series of connectivity and authorization tests to validate your settings

This is only need to be done once and theo will use those settings from that moment. If you ever need to reset this configuration, `reset-vangogh-connection` can help with that.

## Installing native versions with theo

Basic usage of theo comes down to the following commands:

    - `theo install <gog-game-id>` - will install current OS version of a game
    - `theo run <gog-game-id>` - will run installed version of a game
    - `theo uninstall <gog-game-id> -force` - will uninstall a game

More helpful commands:

    - `theo list-installed` - will print all currently installed games
    - `theo reveal-installed` - will open the directory containing game installation

## Installing Windows versions on macOS, Linux with theo

theo provides a set of commands to manage Windows versions on macOS, Linux using [WINE](http://winehq.org):

    - `theo wine-install <gog-game-id>` - will install Windows version of a game
    - `theo wine-run <gog-game-id>` - will run installed version of a game
    - `theo wine-uninstall <gog-game-id> -force` - will uninstall a game

This functionality depends on:

    - macOS: A version of [CrossOver](https://www.codeweavers.com/crossover) purchased and licensed for the current user is required to use WINE-related functionality
    - Linux: theo will use [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) and [GE-Proton](https://github.com/GloriousEggroll/proton-ge-custom). You don't need to do anything, theo will get the latest versions for you automatically as needed

More helpful commands:

    - `theo list-wine-installed` - will print all currently installed Windows versions

## Advanced usage scenarios

### Using specific language version

theo allows you to specify [language](https://en.wikipedia.org/wiki/List_of_ISO_639_language_codes) version to use, when available on GOG and setup on vangogh:

    - `theo install <gog-game-id> -lang-code ja` - will install Japanese version of a game (if it's available)

You need to specify language code for every command (e.g. `run`, `uninstall`, `reveal-installed`).

### Environment variables for Windows version

theo helps you manage Windows versions installations and [WINE](http://winehq.org) options with environmental variables:
    
    - `theo set-prefix-env <gog-game-id> -env VAR1=VAL1 VAR2=VAL2` - will set VAR1 and VAR2 environmental variables for this game installation and those values (VAL1 and VAL2) will be used for any commands under those installations (e.g. `wine-run`)

More helpful commands:

    - `theo default-prefix-env <gog-game-id>` - will reset all environment variables to default values for the current operating system
    - `theo delete-prefix-env <gog-game-id>` - will completely removed all environment variables (even default ones) for this game installation
    - `theo list-prefix-env` - will print all environment variables for every game installed with theo

NOTE: Default environment variables set for each operating systems are listed here: [cli/default_prefix_env.go](https://github.com/arelate/theo/blob/main/cli/default_prefix_env.go#L12)

### Steam shortcuts

theo will automatically add Steam shortcuts (when Steam installation is detected) and will include artwork provided by vangogh. Steam shortcuts are added during `install` and removed during `uninstall`.

While this functionality has been designed around Steam Deck, it works equally well for Steam Big Picture mode on a desktop operating system.

NOTE: Steam shortcuts are added for all users currently logged into Steam under current operating system user account.

Additionally you can use the following commands to manage Steam shortcuts:

    - `theo add-steam-shortcut <gog-game-id>` - will add Steam shortcut for a game
    - `theo remove-steam-shortcut <gog-game-id>` - will remove Steam shortcut for a game
    - `theo list-steam-shortcuts` - will list all existing Steam shortcuts

### Retina mode for Windows installations on macOS

theo provides `mod-prefix-retina` command to set Retina (high DPI). See more information [here](https://gitlab.winehq.org/wine/wine/-/wikis/Commands/winecfg#screen-resolution-dpi-setting). To revert this change use the command with a `revert` flag, e.g.:
    
    - `theo mod-prefix-retina <gog-game-id> -revert`

### Where is theo data stored?

theo stores all state under the current user data directory:

    - on Linux that is ~/.local/share/theo
    - on macOS that is ~/Library/Application Support/theo

## Why theo?

`Theodorus van Gogh 1 May 1857 – 25 January 1891) was a Dutch art dealer and the younger brother of Vincent van Gogh. Known as Theo, his support of his older brother's artistic ambitions and well-being allowed Vincent to devote himself entirely to painting. As an art dealer, Theo van Gogh played a crucial role in introducing contemporary French art to the public.`

## Windows support in theo

NOTE: Implementing (native installations) Windows support in the future should be relatively straightforward - a good starting point would be adding actual implementations for stub functions in [cli/windows_support.go](https://github.com/arelate/theo/blob/main/cli/windows_support.go) and then testing assumptions - e.g. [data/user_dirs.go](https://github.com/arelate/theo/blob/main/data/user_dirs.go).