# Universal ASDF Plugin 🚀

> ⚠️ **Note :** Cette traduction a été réalisée par traduction automatique. Si vous remarquez des inexactitudes, n'hésitez pas à soumettre une pull request avec des corrections.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**Traductions 🌐 :** [English](../README.md) • [Українська](./README.UA.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md) • [日本語](./README.JA.md)

Une collection unifiée de plugins [asdf](https://asdf-vm.com) écrits en Go, remplaçant les plugins bash traditionnels par un binaire unique, testé et maintenable.

## Pourquoi ❓ ?

- 🔐 **Sécurité** — Des plugins bash éparpillés dans de nombreux dépôts augmentent la surface d'attaque potentielle
- ✅ **Fiabilité** — Go facilite l'écriture de tests et la livraison de builds reproductibles
- 🧰 **Maintenance** — Une seule base de code pour plus de 60 outils plutôt que de nombreux plugins séparés au comportement « fourre-tout »

## Démarrage rapide 🚀

```bash
# 1. Téléchargez la dernière version
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Ou installez via Go (nécessite Go 1.24+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Installez asdf (gestionnaire de versions)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Configurez votre shell (ajoutez à ~/.bashrc, ~/.zshrc, etc.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Redémarrez votre shell, puis installez tous les plugins
universal-asdf-plugin install-plugin
```

Après la configuration, gérez vos outils avec asdf :

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Utilisation 🧪

```bash
# Lister les versions disponibles
universal-asdf-plugin list-all <outil>

# Installer une version spécifique
universal-asdf-plugin install <outil> <version>

# Obtenir la dernière version stable
universal-asdf-plugin latest-stable <outil>

# Afficher l'aide pour un outil
universal-asdf-plugin help <outil>

# Mettre à jour .tool-versions vers les dernières versions
universal-asdf-plugin update-tool-versions
```

## Développement 🛠️

### Prérequis

- Go 1.24+
- Docker (pour le dev container)

### Pour commencer

```bash
# Clonez le dépôt
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Ouvrez dans VS Code avec Dev Container
code universal-asdf-plugin.code-workspace

# Construire en local
./scripts/build.sh
```

### Exécuter les tests

```bash
# Mettre à jour les goldenfiles
./scripts/test.sh --update

# Exécuter tous les tests en téléchargeant les paquets réels
./scripts/test.sh --online

# Exécuter tous les tests de fumée avec des serveurs mock
./scripts/test.sh

# Exécuter les tests de mutation
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Vérification orthographique
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# puis inspectez le dictionnaire dans le fichier .code-workspace
```

## Licence 📄

Copyright 2025 Sumicare

En utilisant ce projet, vous acceptez les [Conditions d'utilisation](./OSS_TERMS.FR.md) de Sumicare OSS.

Sous licence [Apache License, Version 2.0](../LICENSE).
