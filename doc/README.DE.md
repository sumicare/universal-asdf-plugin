# Universal ASDF Plugin 🚀

> ⚠️ **Hinweis:** Diese Übersetzung wurde maschinell erstellt. Wenn Sie Ungenauigkeiten bemerken, erstellen Sie bitte einen Pull Request mit Korrekturen.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Übersetzungen 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

Eine einheitliche Sammlung von [asdf](https://asdf-vm.com)-Plugins, geschrieben in Go, die traditionelle Bash-Skript-Plugins durch eine einzelne, getestete und wartbare Binärdatei ersetzt.

## Warum ❓

- 🔐 **Sicherheit** — Traditionelle Bash-Plugins, die über viele Repositories verteilt sind, vergrößern die potenzielle Angriffsfläche
- ✅ **Zuverlässigkeit** — Go erleichtert das Schreiben von Tests und das Erstellen reproduzierbarer Builds
- 🧰 **Wartung** — Eine einzige Codebasis für 60+ Tools statt vieler separat gepflegter Plugins mit "Kitchen-Sink"-Ansatz

## Schnellstart 🚀

```bash
# 1. Laden Sie die neueste Version herunter
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Oder über Go installieren (erfordert Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Installieren Sie asdf (Versionsmanager)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Konfigurieren Sie Ihre Shell (zu ~/.bashrc, ~/.zshrc usw. hinzufügen)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Starten Sie Ihre Shell neu, dann installieren Sie alle Plugins
universal-asdf-plugin install-plugin
```

Nach der Einrichtung verwalten Sie Ihre Tools mit asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Verwendung 🧪

```bash
# Verfügbare Versionen auflisten
universal-asdf-plugin list-all <tool>

# Eine bestimmte Version installieren
universal-asdf-plugin install <tool> <version>

# Die neueste stabile Version abrufen
universal-asdf-plugin latest-stable <tool>

# Hilfe für ein Tool anzeigen
universal-asdf-plugin help <tool>

# .tool-versions auf die neuesten Versionen aktualisieren
universal-asdf-plugin update-tool-versions
```

## Entwicklung 🛠️

### Voraussetzungen

- Go 1.24+
- Docker (für Dev Container)

### Erste Schritte

```bash
# Repository klonen
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# In VS Code mit Dev Container öffnen
code universal-asdf-plugin.code-workspace

# Lokal bauen
./scripts/build.sh
```

### Tests ausführen

```bash
# Goldenfiles aktualisieren
./scripts/test.sh --update

# Alle Tests ausführen und echte Pakete herunterladen
./scripts/test.sh --online

# Alle Smoke-Tests mit gemockten Servern ausführen
./scripts/test.sh

# Mutation-Tests ausführen
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Rechtschreibprüfung
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# anschließend das Wörterbuch in der .code-workspace-Datei prüfen
```

## Lizenz 📄

Copyright 2025 Sumicare

Durch die Nutzung dieses Projekts stimmen Sie den [Nutzungsbedingungen](./OSS_TERMS.DE.md) von Sumicare OSS zu.

Lizenziert unter der [Apache License, Version 2.0](../LICENSE).
