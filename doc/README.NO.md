# Universal ASDF Plugin 🚀

> ⚠️ **Merk:** Denne oversettelsen er maskinoversatt. Hvis du oppdager unøyaktigheter, vennligst opprett en pull request med rettelser.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Oversettelser 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

En samlet samling av [asdf](https://asdf-vm.com)-plugins skrevet i Go, som erstatter tradisjonelle bash-skript-plugins med en enkelt, testet og vedlikeholdbar binærfil.

## Hvorfor ❓

- 🔐 **Sikkerhet** — Tradisjonelle bash-plugins som ligger spredt i mange repoer øker den potensielle angrepsflaten
- ✅ **Pålitelighet** — Go gjør det enklere å skrive tester og få reproduserbare bygg
- 🧰 **Vedlikehold** — Én felles kodebase for 60+ verktøy i stedet for mange separate «alt mulig»-plugins

## Hurtigstart 🚀

```bash
# 1. Last ned den nyeste versjonen
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Eller installer via Go (krever Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Installer asdf (versjonsbehandler)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Konfigurer skallet (legg til i ~/.bashrc, ~/.zshrc osv.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Start skallet på nytt, deretter installer alle plugins
universal-asdf-plugin install-plugin
```

Etter oppsett, administrer verktøyene dine med asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Bruk 🧪

```bash
# List tilgjengelige versjoner
universal-asdf-plugin list-all <verktøy>

# Installer en spesifikk versjon
universal-asdf-plugin install <verktøy> <versjon>

# Hent den nyeste stabile versjonen
universal-asdf-plugin latest-stable <verktøy>

# Vis hjelp for et verktøy
universal-asdf-plugin help <verktøy>

# Oppdater .tool-versions til de nyeste versjonene
universal-asdf-plugin update-tool-versions
```

## Utvikling 🛠️

### Forutsetninger

- Go 1.24+
- Docker (for dev container)

### Kom i gang

```bash
# Klon repositoriet
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Åpne i VS Code med Dev Container
code universal-asdf-plugin.code-workspace

# Bygg lokalt
./scripts/build.sh
```

### Kjøre tester

```bash
# Oppdater goldenfiles
./scripts/test.sh --update

# Kjør alle tester og last ned faktiske pakker
./scripts/test.sh --online

# Kjør alle smoke-tester med mockede servere
./scripts/test.sh

# Kjør mutation-tester
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Stavekontroll
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# sjekk deretter ordboken i .code-workspace-filen
```

## Lisens 📄

Copyright 2025 Sumicare

Ved å bruke dette prosjektet godtar du Sumicare OSS [Bruksvilkår](./OSS_TERMS.NO.md).

Lisensiert under [Apache License, Version 2.0](../LICENSE).
