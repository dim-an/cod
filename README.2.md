# Cod - projektdokumentation

Det här dokumentet är en teknisk genomgång av projektet i denna repo. Den befintliga `README.md` beskriver användning och installation på hög nivå. Denna fil fokuserar på kodens struktur, körflöden, build, tester och vilka artefakter som skapas.

## Sammanfattning

`cod` är ett Go-baserat CLI-verktyg och en shell-completion-daemon för `bash`, `fish` och `zsh`.

Grundidén är:

1. Shell-integration laddas via `cod init <pid> <shell>`.
2. `cod` startar eller återanvänder en daemon.
3. Shell hooks anropar interna `cod api ...`-kommandon efter körda kommandon och vid completion.
4. När ett lyckat, enkelt kommando innehåller `--help` kan `cod` lära sig kommandots options.
5. Daemonen kör help-kommandot, parser outputen, sparar completions i SQLite och genererar shell-script för uppdaterad completion.

## Teknisk stack

- Språk: Go.
- Modul: `github.com/dim-an/cod`.
- CLI-parser: `github.com/alecthomas/kingpin/v2`.
- Databas: SQLite via `github.com/ncruces/go-sqlite3`.
- Konfiguration: TOML via `github.com/pelletier/go-toml`.
- Tester: Go `testing` plus `github.com/stretchr/testify`.
- IPC: Unix domain socket med JSON-meddelanden.

`go.mod` anger `go 1.26.0`. Verifierad lokal miljö vid genomgång: `go1.26.3 darwin/arm64`.

## CLI och kommandon

Huvudkommandon definieras i `main.go`.

Publika kommandon:

- `learn <subject>...`: lär in completions från ett help-kommando.
- `list [--format table|plain] [selector...]` eller `ls`: listar sparade kommandon. Default är en tabell med beskrivning och antal completions; `plain` behåller det äldre `id<TAB>kommando`-formatet.
- `show <selector...>`: visar detaljer för sparade kommandon, inklusive executable path, kommandobeskrivning och completion-beskrivningar.
- `remove <selector...>` eller `rm`: tar bort sparade kommandon.
- `update <selector...>`: kör om tidigare help-kommandon och uppdaterar completions.
- `init <pid> <shell>`: skriver shell-init-script för bash, fish eller zsh och attachar daemonen.
- `example-config [--create]`: skriver exempelkonfiguration till stdout eller skapar configfilen.
- `daemon [--foreground]`: startar daemonen.

Dolda interna API-kommandon:

- `api attach`: attachar en shellprocess till daemonen.
- `api postexec`: analyserar senast körda kommando efter prompt.
- `api poll-updates`: returnerar shell-script för completion-uppdateringar.
- `api complete-words`: returnerar completion-kandidater.
- `api list-clients`: listar attachade shellklienter.
- `api forked-daemon`: intern daemon-child vid daemonisering.
- `api bash-clean-completions`: rensar bash-completion-definitioner för ett kommando.

## Runtime-flöde

### Initiering

`cod init <pid> <shell>` gör följande:

1. Skapar logg- och run-kataloger via `server.Configuration`.
2. Startar daemon om den inte redan kör.
3. Skickar `AttachRequest` till daemonen med shell, pid och sökväg till `cod`-binären.
4. Hämtar `InitScriptResponse` från daemonen.
5. Skriver shell-script till stdout, som användaren normalt sourcar i shell-initfilen.

### Daemon

Daemonen finns i `daemon.go` och `server/server.go`.

- `daemonMain` kan köras i foreground eller daemonisera.
- `daemonProc` byter arbetskatalog till `/`, tar lock på lockfilen, tar bort gammal socket och startar servern.
- Servern lyssnar på Unix socket från `Configuration.GetSocketFile`.
- Varje request är JSON och routas efter request-namn i `serverImpl.handleRequest`.
- Daemonen håller en karta över attachade shellprocesser och avslutar när sista shellprocessen detachas eller dör.

### Inlärning

Inlärning sker manuellt via `cod learn ...` eller automatiskt efter `--help`-kommandon.

Viktiga steg:

1. Kommandot canonicaliseras till absolut executable path.
2. Daemonen kör help-kommandot med sparad miljö och arbetskatalog.
3. Output parseras av `parse_doc.ParseHelp`.
4. Help page, kommandobeskrivning, completions och completion-beskrivningar sparas i SQLite.
5. Attachade shellklienter markeras för uppdatering.
6. `api poll-updates` returnerar reset- och add-script för berörda kommandon.

### Completion

Vid completion anropar shell-scriptet:

```sh
cod api complete-words -- <pid> <c-word> <words...>
```

Servern:

1. Hämtar completions för executable path och flagg-prefix från SQLite.
2. Kontrollerar eventuell subcommand-kontext.
3. Returnerar matchande flags, en per rad.

## Kodindex

### Rotpaketet `main`

- `main.go`: CLI-definition, versionssträng och dispatch till kommandon.
- `commands.go`: implementation av `learn`, `list`, `remove`, `update`, `init`, `example-config` och interna API-kommandon.
- `daemon.go`: daemonisering, forkad daemon, lockfil, signalering och loggtrimning.
- `application.go`: lat laddning av `server.Configuration` och `server.Client`.
- `api_attach.go`: attach-flöde som kan starta daemonen innan attach.
- `example_configuration.go`: texten som `cod example-config` skriver ut.
- `ui.go`: interaktiv terminal-UI för ja/nej-frågor samt färgad output.
- `release.py`: installationshjälp som flyttar byggd binär till `~/.local/bin/cod`, dödar daemon och re-attachar klienter.
- `cod.plugin.zsh`: zsh-plugin som sourcar `cod init $$ zsh` om `cod` finns.

### `server`

Ansvarar för daemonens serverdel, klient, konfiguration och wire-protokoll.

- `configuration.go`: härledda XDG-sökvägar för config, data, socket, lockfil, logg och databas.
- `client.go`: Unix-socket-klient som skickar JSON-requests.
- `request.go`: request/response-typer och JSON-marshalling.
- `server.go`: server-loop, request-routing och all central business logic.
- `user_configuration.go`: laddar TOML-regler och bestämmer policy per executable.
- `errors.go`: serialiserbara remote errors och felkoder.

### `datastore`

Ansvarar för persistens.

- `data.go`: domänmodeller som `Command`, `Completion`, `HelpPage`, `HelpPageInfo`, `Policy` och path-canonicalisering.
- `sqlitedb.go`: SQLite-schema, transaktioner, CRUD och merge-logik för help pages.
- Databasschemat har tabellerna `HelpPage` och `Completion`.
- Aktuell schemaversion är v2. Migrationen från v1 lägger till tomma `Description`-fält och bevarar befintliga kommandon/completions.
- `HelpPage` lagrar executable path, helptext-checksum, command-args-checksum, command JSON, policy och beskrivning.
- `Completion` lagrar flagga, beskrivning och optional kontext per help page.

### `parse_doc`

Ansvarar för att omvandla help-text till completions.

- `parse_help.go`: väljer parserordning och bygger `datastore.HelpPage`.
- `argparse.go`: parser specialiserad för Python argparse, inklusive subcommands.
- `parse_default.go`: generisk parser för flags och enklare subcommand-fall.
- `textutil.go`: hjälpstrukturer för text, paragrafer och indentering.

Parserordning:

1. argparse-parser.
2. default-parser.

### `shells`

Ansvarar för shellspecifik integration och parsing av enkla kommandorader.

- `shells.go`: genererar bash-, fish- och zsh-script.
- `tokenize.go`: tokeniserar shell-kommandon och markerar "scary" syntax.
- `parse.go`: accepterar endast enkla kommandon och initiala variabelassignments.
- `quote.go`: shell-quoting.
- `remove_completions.go`: filtrerar bort bash-completions för specifikt kommando.
- `asciitable`: små ASCII-konstanter.

### `util`

Gemensamma hjälpfunktioner:

- `find_path.go`: PATH-sökning och miljövariabelhämtning.
- `selector.go`: selectors/globs för list, remove och update.
- `util.go`: kataloghantering, hashing, warnings och sort/uniq.

### `test`

End-to-end-tester som kör byggd `cod`-binär.

- `lib.go`: `Workbench` som skapar temporära XDG-kataloger, startar fake-shell-processer och kör `cod`.
- `binaries/`: små Python-program som används som testkommandon med olika help-output.
- `learn_test.go`: manuell inlärning, path-upplösning, merge och subcommand-context.
- `list_test.go`: listning, selector-filter och remove.
- `poll_update_test.go`: shell update-script efter ändrade completions.
- `update_test.go`: update, merge, no-op och borttag vid trasigt kommando.
- `smoke_test.go`: daemon-start och attach/detach-livscykel.
- `example_configuration_test.go`: validerar exempelkonfigurationen.

## Build

Rekommenderat build-kommando:

```sh
make build
```

Det kör:

```sh
go build -o cod -ldflags "-X main.GitSha=`git rev-parse HEAD`"
```

Förväntat resultat:

- En körbar binär `./cod` skapas i repo-roten.
- Binären innehåller aktuell git sha i `main.GitSha`.
- `cod` är ignorerad i `.gitignore`.

Alternativt kan man köra direkt:

```sh
go build -o cod
```

men då sätts inte `GitSha` på samma sätt som i Makefile.

## Install

```sh
make install
```

Det kör först build och sedan `python release.py`.

`release.py` förväntar sig att `./cod` finns, flyttar den till `~/.local/bin/cod`, stoppar eventuell körande `cod`-daemon och försöker attacha tidigare klienter igen.

## Tester

Rekommenderat testkommando:

```sh
make test
```

Det gör:

1. `make build`.
2. `COD_TEST_BINARY="<repo>/cod" go test ./...`.

Verifierat resultat 2026-05-17:

```text
ok   github.com/dim-an/cod/datastore
ok   github.com/dim-an/cod/parse_doc
ok   github.com/dim-an/cod/shells
ok   github.com/dim-an/cod/test
ok   github.com/dim-an/cod/util
```

Paket utan testfiler:

- `github.com/dim-an/cod`
- `github.com/dim-an/cod/server`
- `github.com/dim-an/cod/shells/asciitable`

### Testtyper

Rena pakettester:

- `datastore/*_test.go`: path-canonicalisering, command context, SQLite CRUD, merge och listning.
- `parse_doc/*_test.go`: default parser, argparse parser, textutil och specifika help-exempel.
- `shells/*_test.go`: tokenisering, quoting och bash completion cleanup.
- `util/selector_test.go`: selector/glob-matchning.

End-to-end-tester:

- Ligger i `test/`.
- Bygger inte själva binären, utan kräver `COD_TEST_BINARY`.
- `make test` sätter `COD_TEST_BINARY` till den lokalt byggda `./cod`.
- Startar fake-shell-processer med `sleep`.
- Kör `cod init`, `cod learn`, `cod list`, `cod update` och interna `cod api ...`-kommandon.

### Vad testerna skapar

Vid körning skapar `test.Workbench`:

- Temporär systemkatalog: `/tmp/cod-test-*`.
- Symlinkad arbetskatalog för kortare socket-sökvägar.
- Per-test-kataloger under `test-data/<TestNamn>` inuti den temporära strukturen.
- Temporära XDG-kataloger:
  - `XDG_CONFIG_HOME=<temp>/config`
  - `XDG_DATA_HOME=<temp>/data`

Runtime skapar sedan under temporär `XDG_DATA_HOME`:

- `cod/db.sqlite3`: SQLite-databas.
- `cod/var/cod.sock`: Unix socket.
- `cod/var/cod.lock`: lockfil med daemon-pid.
- `cod/log/cod.YYYY-MM-DD.log`: daemonlogg.

Vid lyckade tester tas temporär testdata bort. Vid misslyckade tester försöker `Workbench.Close` skriva ut loggfilerna för felsökning.

## Runtime-filer vid normal användning

Konfiguration:

- Om `XDG_CONFIG_HOME` saknas används `~/.config`.
- User config ligger i `$XDG_CONFIG_HOME/cod/config.toml`.

Data:

- Om `XDG_DATA_HOME` saknas används `~/.local/share`.
- Databas: `$XDG_DATA_HOME/cod/db.sqlite3`.
- Socket och lock: `$XDG_DATA_HOME/cod/var/`.
- Loggar: `$XDG_DATA_HOME/cod/log/`.

## Konfiguration och policy

`cod example-config` skriver ett TOML-exempel.

Viktiga inställningar:

- `command-execution-timeout`: timeout i millisekunder för help-kommandon. Default är 1000.
- `[[rule]]`: regler per executable selector.
- `policy` kan vara:
  - `ask`: fråga användaren.
  - `trust`: lär automatiskt.
  - `ignore`: ignorera.

Selectorformer stöds via `util.CompileSelector`, bland annat:

- exakt absolut sökväg.
- `/path/*`.
- `/path/**`.
- basename, till exempel `git`.
- `~/...` expanderas mot HOME.

## Viktiga beteenden

- Bara enkla kommandorader analyseras automatiskt. Pipes, redirections och annan komplex shellsyntax ignoreras.
- `--help` efter `--` räknas inte som help-signal, eftersom parsing avbryts vid `--`.
- Executables canonicaliseras till absolut path innan completions sparas eller slås upp.
- Completions filtreras både på prefix och eventuell subcommand-kontext.
- Om ett uppdaterat help-kommando inte längre går att köra tar `update` bort den gamla help page-posten.
- Daemonen avslutas när sista attachade shellprocessen försvinner.

## Kända observationspunkter

- `Makefile` har separata `cd ${THISDIR}`-rader. Eftersom Make kör varje rad i eget shell påverkar de inte nästa rad. Från repo-roten fungerar kommandona ändå som förväntat.
- `go test ./...` behöver kunna skriva till Go build-cache. I sandboxad miljö kan det kräva extra tillåtelse.
- Integrationstesterna använder Unix-processer, signaler, sockets och temporära XDG-kataloger, vilket gör dem mer miljökänsliga än rena pakettester.
