# Universal ASDF Plugin 🚀

> ⚠️ **Poznámka:** Tento překlad byl vytvořen strojově. Pokud si všimnete nepřesností, vytvořte prosím pull request s opravami.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Překlady 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

Sjednocená kolekce pluginů [asdf](https://asdf-vm.com) napsaných v Go, nahrazující tradiční bash skripty jediným testovaným a udržovatelným binárním souborem.

## Proč ❓

- 🔐 **Bezpečnost** — Tradiční bash pluginy rozptýlené v různých repozitářích zvětšují potenciální útočnou plochu
- ✅ **Spolehlivost** — Go usnadňuje psaní testů a vytváření reprodukovatelných buildů
- 🧰 **Údržba** — Jediná kódová základna pro 60+ nástrojů místo mnoha samostatných pluginů s „kitchen-sink“ přístupem

## Rychlý start 🚀

```bash
# 1. Stáhněte nejnovější verzi
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Nebo nainstalujte pomocí Go (vyžaduje Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Nainstalujte asdf (správce verzí)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Nakonfigurujte shell (přidejte do ~/.bashrc, ~/.zshrc atd.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Restartujte shell, poté nainstalujte všechny pluginy
universal-asdf-plugin install-plugin
```

Po nastavení spravujte nástroje pomocí asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Použití 🧪

```bash
# Seznam dostupných verzí
universal-asdf-plugin list-all <nástroj>

# Instalace konkrétní verze
universal-asdf-plugin install <nástroj> <verze>

# Získání nejnovější stabilní verze
universal-asdf-plugin latest-stable <nástroj>

# Zobrazení nápovědy pro nástroj
universal-asdf-plugin help <nástroj>

# Aktualizace .tool-versions na nejnovější verze
universal-asdf-plugin update-tool-versions
```

## Vývoj 🛠️

### Předpoklady

- Go 1.24+
- Docker (pro dev container)

### Začínáme

```bash
# Naklonujte repozitář
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Otevřete ve VS Code s Dev Container
code universal-asdf-plugin.code-workspace

# Sestavte lokálně
./scripts/build.sh
```

### Spouštění testů

```bash
# Aktualizace goldenfiles
./scripts/test.sh --update

# Spuštění všech testů se stahováním skutečných balíčků
./scripts/test.sh --online

# Spuštění všech smoke testů s mockovanými servery
./scripts/test.sh

# Spuštění mutation testů
./scripts/mutation-test.sh

# Lintování
./scripts/lint.sh

# Kontrola pravopisu
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# poté zkontrolujte slovník v souboru .code-workspace
```

## Licence 📄

Copyright 2025 Sumicare

Používáním tohoto projektu souhlasíte s [Podmínkami použití](./OSS_TERMS.CS.md) Sumicare OSS.

Licencováno pod [Apache License, Version 2.0](../LICENSE).
