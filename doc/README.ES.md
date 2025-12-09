# Universal ASDF Plugin 🚀

> ⚠️ **Nota:** Esta traducción se ha generado mediante traducción automática. Si encuentra errores o frases extrañas, por favor envíe un pull request con las correcciones.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Traducciones 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [한국어](./README.KO.md) • [日本語](./README.JA.md)

Colección unificada de plugins de [asdf](https://asdf-vm.com) escritos en Go, que sustituye los plugins tradicionales basados en bash por un único binario probado y fácil de mantener.

## ¿Por qué ❓

- 🔐 **Seguridad** — Los plugins bash repartidos por diferentes repositorios representan una superficie de ataque real.
- ✅ **Fiabilidad** — Go facilita la escritura de tests y la obtención de builds reproducibles.
- 🧰 **Mantenibilidad** — Una sola base de código para más de 60 herramientas, en lugar de muchos plugins separados con convenciones heterogéneas.

## Inicio rápido 🚀

```bash
# 1. Descargar la última versión
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# O instalar vía Go (requiere Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Bootstrap de asdf (instala el propio gestor de versiones asdf)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Configurar la shell (añadir a ~/.bashrc, ~/.zshrc, etc.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Reiniciar la shell y luego instalar los plugins necesarios
universal-asdf-plugin install-plugin
```

Después de la configuración, gestione sus herramientas mediante asdf:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Uso 🧪

```bash
# Listar versiones disponibles
universal-asdf-plugin list-all <tool>

# Instalar una versión concreta
universal-asdf-plugin install <tool> <version>

# Obtener la última versión estable
universal-asdf-plugin latest-stable <tool>

# Mostrar la ayuda de una herramienta
universal-asdf-plugin help <tool>

# Actualizar .tool-versions a las versiones más recientes
universal-asdf-plugin update-tool-versions
```

## Desarrollo 🛠️

### Requisitos previos

- Go 1.24+
- Docker (para Dev Container)

### Primeros pasos

```bash
# Clonar el repositorio
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Abrir en VS Code con Dev Container
code universal-asdf-plugin.code-workspace

# Compilar localmente
./scripts/build.sh
```

### Ejecutar tests

```bash
# Actualizar ficheros golden
./scripts/test.sh --update

# Ejecutar todos los tests descargando paquetes reales
./scripts/test.sh --online

# Ejecutar todos los smoke tests con servidores mock
./scripts/test.sh

# Ejecutar tests de mutación
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Comprobación ortográfica
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# después revise el diccionario en el fichero .code-workspace
```

## Licencia 📄

Copyright 2025 Sumicare

Al utilizar este proyecto, usted acepta los [Términos de uso](./OSS_TERMS.ES.md) de Sumicare OSS.

Este proyecto está licenciado bajo la [Apache License, Version 2.0](../LICENSE).
