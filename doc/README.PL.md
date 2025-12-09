# Universal ASDF Plugin 🚀

> ⚠️ **Uwaga:** To tłumaczenie zostało wykonane maszynowo. Jeśli zauważysz nieścisłości, prosimy o utworzenie pull requesta z poprawkami.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Tłumaczenia 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

Zunifikowana kolekcja wtyczek [asdf](https://asdf-vm.com) napisanych w Go, zastępująca tradycyjne wtyczki bash pojedynczym, przetestowanym i łatwym w utrzymaniu plikiem binarnym.

## Dlaczego ❓

- 🔐 **Bezpieczeństwo** — Tradycyjne wtyczki bash rozproszone po wielu repozytoriach zwiększają potencjalną powierzchnię ataku
- ✅ **Niezawodność** — Go ułatwia pisanie testów i budowanie powtarzalnych kompilacji
- 🧰 **Utrzymanie** — Jedna baza kodu dla 60+ narzędzi zamiast wielu osobnych wtyczek tworzonych w stylu „kitchen-sink”

## Szybki start 🚀

```bash
# 1. Pobierz najnowszą wersję
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Lub zainstaluj za pomocą Go (wymaga Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Zainstaluj asdf (menedżer wersji)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Skonfiguruj powłokę (dodaj do ~/.bashrc, ~/.zshrc itp.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Uruchom ponownie powłokę, następnie zainstaluj wszystkie wtyczki
universal-asdf-plugin install-plugin
```

Po konfiguracji zarządzaj narzędziami przez asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Użycie 🧪

```bash
# Lista dostępnych wersji
universal-asdf-plugin list-all <narzędzie>

# Zainstaluj konkretną wersję
universal-asdf-plugin install <narzędzie> <wersja>

# Pobierz najnowszą stabilną wersję
universal-asdf-plugin latest-stable <narzędzie>

# Pokaż pomoc dla narzędzia
universal-asdf-plugin help <narzędzie>

# Zaktualizuj .tool-versions do najnowszych wersji
universal-asdf-plugin update-tool-versions
```

## Rozwój 🛠️

### Wymagania wstępne

- Go 1.24+
- Docker (dla dev container)

### Rozpoczęcie pracy

```bash
# Sklonuj repozytorium
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Otwórz w VS Code z Dev Container
code universal-asdf-plugin.code-workspace

# Zbuduj lokalnie
./scripts/build.sh
```

### Uruchamianie testów

```bash
# Zaktualizuj goldenfiles
./scripts/test.sh --update

# Uruchom wszystkie testy z pobieraniem rzeczywistych pakietów
./scripts/test.sh --online

# Uruchom wszystkie testy typu smoke z mockowanymi serwerami
./scripts/test.sh

# Uruchom testy mutacyjne
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Sprawdzanie pisowni
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# następnie sprawdź słownik w pliku .code-workspace
```

## Licencja 📄

Copyright 2025 Sumicare

Korzystając z tego projektu, zgadzasz się na [Warunki użytkowania](./OSS_TERMS.PL.md) Sumicare OSS.

Licencjonowane na podstawie [Apache License, Version 2.0](../LICENSE).
