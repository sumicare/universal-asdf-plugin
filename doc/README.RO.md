# Universal ASDF Plugin 🚀

> ⚠️ **Notă:** Această traducere a fost realizată prin traducere automată. Dacă observați inexactități, vă rugăm să creați un pull request cu corecturi.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Traduceri 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

O colecție unificată de plugin-uri [asdf](https://asdf-vm.com) scrise în Go, înlocuind plugin-urile tradiționale bash cu un singur binar testat și ușor de întreținut.

## De ce ❓?

- 🔐 **Securitate** — Plugin-urile bash tradiționale împrăștiate în mai multe depozite măresc suprafața de atac potențială
- ✅ **Fiabilitate** — Go simplifică scrierea testelor și obținerea unor build-uri reproductibile
- 🧰 **Întreținere** — O singură bază de cod pentru 60+ instrumente, în locul multor plugin-uri separate cu o abordare de tip „kitchen-sink”

## Start rapid 🚀

```bash
# 1. Descărcați ultima versiune
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Sau instalați prin Go (necesită Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Instalați asdf (managerul de versiuni)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Configurați shell-ul (adăugați în ~/.bashrc, ~/.zshrc etc.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Reporniți shell-ul, apoi instalați toate plugin-urile
universal-asdf-plugin install-plugin
```

După configurare, gestionați instrumentele cu asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Utilizare 🧪

```bash
# Listați versiunile disponibile
universal-asdf-plugin list-all <instrument>

# Instalați o versiune specifică
universal-asdf-plugin install <instrument> <versiune>

# Obțineți ultima versiune stabilă
universal-asdf-plugin latest-stable <instrument>

# Afișați ajutorul pentru un instrument
universal-asdf-plugin help <instrument>

# Actualizați .tool-versions la ultimele versiuni
universal-asdf-plugin update-tool-versions
```

## Dezvoltare 🛠️

### Cerințe preliminare

- Go 1.24+
- Docker (pentru dev container)

### Primii pași

```bash
# Clonați depozitul
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Deschideți în VS Code cu Dev Container
code universal-asdf-plugin.code-workspace

# Compilați local
./scripts/build.sh
```

### Rulare teste

```bash
# Actualizați fișierele golden
./scripts/test.sh --update

# Rulați toate testele cu descărcarea pachetelor reale
./scripts/test.sh --online

# Rulați toate smoke testele cu servere mock
./scripts/test.sh

# Rulați testele de mutație
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Verificare ortografică
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# apoi inspectați dicționarul din fișierul .code-workspace
```

## Licență 📄

Copyright 2025 Sumicare

Prin utilizarea acestui proiect, sunteți de acord cu [Termenii de utilizare](./OSS_TERMS.RO.md) Sumicare OSS.

Licențiat sub [Apache License, Version 2.0](../LICENSE).
