![](https://img.shields.io/badge/GO-passing-green?style=for-the-badge&logo=Go)

Cod is a completion daemon for ```bash```, ```fish```, and ```zsh```.

It detects usage of ```--help``` commands, parses their output, and generates
auto-completions for your shell.

![https://asciinema.org/a/h0SrrNvZVcqoSM4DNyEUrGtQh](file:https://asciinema.org/a/h0SrrNvZVcqoSM4DNyEUrGtQh.svg)

# Install

  You can either [download](https://github.com/dim-an/cod/releases) or [build](https://github.com/dim-an/cod/blob/master/README.org#Build) the ```cod``` binary
  for your OS and put it into your ```$PATH```.

  After that, you will need to edit your init script (e.g. ```~/.config/fish/config.fish```, ```~/.zshrc```, ```~/.bashrc```) and add a few lines for
  the daemon to work correctly.

### Bash
   Add the following to your ```~/.bashrc```
   ```bash
   source <(cod init $$ bash)
   ```

### Zsh
   [Make sure](#compsys_init) completion system is initialized.

   Add the following to your ```~/.zshrc```
   ```zsh
   source <(cod init $$ zsh)
   ```
   Or, you can use a plugin manager like zinit:
   ```zsh
   zinit wait lucid for \
     dim-an/cod
   ```

#### <a name="compsys_init"></a> Initializing zsh completion system

  `cod` requires initialized completion system.
  In many cases it is already the case (e.g. if you are using oh-my-zsh or similar framework).

  You can check whether your completion system is already initilized by using `type compdef` command:
  ```
  # Completion system IS initialized
  $ type compdef
  compdef is a shell function from /usr/share/zsh/functions/Completion/compinit

  # Completion system IS NOT initialized
  $ type compdef
  compdef not found
  ```

  If you found that you need to initialize completion system you can do this by:

   - calling `compinit` function in your `.zshrc` before initializing `cod` itself, or
   - executing `compinstall` command from your shell, it will modify `.zshrc` file for you.

  Also check [zsh documentation](https://zsh.sourceforge.io/Doc/Release/Completion-System.html).


### Fish
   Add the following to ```~/.config/fish/config.fish```
   ```fish
   cod init $fish_pid fish | source
   ```

### Fig

As an alternative, you can also install ```cod``` with [Fig](https://fig.io/plugins/other/cod_dim-an) in ```bash```, ```zsh```, or ```fish``` with just one click.

![https://fig.io/plugins/other/cod_dim-an](https://fig.io/badges/install-with-fig.svg)

### Supported shells and operating systems

   - zsh
   ```cod``` is known to work with latest version of ```zsh``` (tested: ```v5.5.1``` and
   ```5.7.1```) on macOS and Linux.

   - bash
   ```cod``` also works with with latest version of ```bash``` (tested: ```4.4.20``` and
   ```v5.0.11```) on Linux.

     Note that default ```bash``` that is bundled with macOS is too old and ```cod```
     doesn't support it.

   - fish
   ```cod``` works with latest version of ```fish``` (tested: ```v3.1.2```) on Linux
   (I didn't have a chance to test it on macOS).


# Building cod
  It is recommended that you have at least [Go v1.19](https://golang.org/dl/) installed on your machine
  ```bash
  git clone https://github.com/dim-an/cod.git
  cd cod
  go build
  ```

  or

  ```bash
  go get -u github.com/dim-an/cod
  ```

# Overview
  Cod checks each command you run in the shell. When cod detects usage of
  ```--help``` flag it asks if you want it to learn this command. If you choose
  to allow cod to learn this command cod will run command itself parse the
  output and generate completions based on the ```--help``` output.

## How cod detects help commands
   Cod performs following checks to decide if command is help invocation:
   1. checks if the ```--help``` flag is used
   2. checks that command is simple i.e. doesn't contain any pipes, file
     descriptor redirections, and other shell magic
   3. checks that command exit code is 0.

   If cod cannot automatically detect that your command is help invocation
   you can use ```learn``` subcommand to learn this command anyway.

## How cod runs help commands
   Cod always uses absolute paths to run programs. (So it finds the binary in
   ```$PATH``` or resolves relative path if required). Arguments other than
   the binary path are left unchanged.

   The current shell environment and current working directory will be
   used.

   If the program is successfully executed, cod will store:
     - the absolute path to binary
     - any used arguments
     - the working directory
     - environment variables
   This info will be used to update command if required (check:
   ```cod help update```).

## How cod parses help output
   ```cod``` has generic parser that works with most help pages and
   recognizes flags (starting with ```-```), while not recognizing subcommands.

   It also has a special parser tuned for [the python argparse library](https://docs.python.org/library/argparse.html)
   that recognizes flags and subcommands.

# Configuration
  Cod will search for the default config file ```$XDG_CONFIG_HOME/cod/config.toml```.

  The config file allows you to specify rules to either ignore or trust specified binaries

  ```cod example-config``` prints an example configuration to stdout.

  ```cod example-config --create``` writes an example config to the default directory of said config file (```$XDG_CONFIG_HOME/cod/config.toml```)

# Data directories
  ```cod``` uses ```$XDG_DATA_HOME/cod``` (default: ```~/.local/share/cod```) to store all
  generated data files.
