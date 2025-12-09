# Universal ASDF Plugin 🚀

> ⚠️ **Примітка:** Цей переклад створено за допомогою машинного перекладу. Якщо ви помітили неточності, будь ласка, створіть pull request з виправленнями.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Переклади 🌐:** [English](../README.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

Уніфікована колекція плагінів [asdf](https://asdf-vm.com), написаних на Go, що замінює традиційні bash-скрипти єдиним протестованим бінарним файлом.

## Чому ❓

- 🔐 **Безпека** — Традиційні bash-плагіни, розкидані по різних репозиторіях, збільшують потенціал для атаки
- ✅ **Надійність** — Go спрощує написання тестів і забезпечує відтворювані збірки
- 🧰 **Підтримка** — Єдина кодова база для 60+ інструментів замість багатьох окремих «кухонних» плагінів

## Швидкий старт 🚀

```bash
# 1. Завантажте останній реліз
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Або встановіть через Go (потрібно Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Встановіть asdf (менеджер версій)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Налаштуйте оболонку (додайте до ~/.bashrc, ~/.zshrc тощо)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Перезапустіть оболонку, потім встановіть усі плагіни
universal-asdf-plugin install-plugin
```

Після налаштування керуйте інструментами через asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Використання 🧪

```bash
# Список доступних версій
universal-asdf-plugin list-all <інструмент>

# Встановити конкретну версію
universal-asdf-plugin install <інструмент> <версія>

# Отримати останню стабільну версію
universal-asdf-plugin latest-stable <інструмент>

# Показати довідку для інструменту
universal-asdf-plugin help <інструмент>

# Оновити .tool-versions до останніх версій
universal-asdf-plugin update-tool-versions
```

## Розробка 🛠️

### Передумови

- Go 1.24+
- Docker (для dev container)

### Початок роботи

```bash
# Клонуйте репозиторій
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Відкрийте у VS Code з Dev Container
code universal-asdf-plugin.code-workspace

# Зібрати локально
./scripts/build.sh
```

### Запуск тестів

```bash
# Оновити goldenfiles
./scripts/test.sh --update

# Запустити всі тести з завантаженням реальних пакетів
./scripts/test.sh --online

# Запустити всі smoke-тести з моканими серверами
./scripts/test.sh

# Запустити mutation-тести
./scripts/mutation-test.sh

# Лінтинг
./scripts/lint.sh

# Перевірка орфографії
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# потім перевірте словник у .code-workspace
```

## Ліцензія 📄

Copyright 2025 Sumicare

Використовуючи цей проєкт, ви погоджуєтесь з [Умовами використання](./OSS_TERMS.UA.md) Sumicare OSS.

Ліцензовано за [Apache License, Version 2.0](../LICENSE).
