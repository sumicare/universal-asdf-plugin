# Universal ASDF Plugin

> ⚠️ **注意：** この翻訳は機械翻訳によって作成されました。不正確な点がございましたら、プルリクエストで修正をお願いいたします。

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**翻訳：** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [中文](./README.ZH.md)

Go で書かれた [asdf](https://asdf-vm.com) プラグインの統合コレクションです。従来の bash スクリプトプラグインを、単一のテスト済みで保守しやすいバイナリに置き換えます。

## なぜこのプロジェクト？

- **セキュリティ** — Bash プラグインは攻撃対象となり得ます。このプロジェクトは信頼を統合します
- **信頼性** — Go は型安全性、テスト、再現可能なビルドを提供します
- **保守性** — 分散したリポジトリの代わりに、60以上のツールを単一のコードベースで管理
- **パフォーマンス** — 並列操作をサポートするネイティブバイナリ

## クイックスタート

```bash
# 1. 最新リリースをダウンロード
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# または Go からインストール（Go 1.24+ が必要）
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. asdf をインストール（バージョンマネージャー）
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. シェルを設定（~/.bashrc、~/.zshrc などに追加）
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. シェルを再起動し、すべてのプラグインをインストール
universal-asdf-plugin install-plugin
```

セットアップ後、asdf でツールを管理：

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## 使用方法

```bash
# 利用可能なバージョンを一覧表示
universal-asdf-plugin list-all <ツール>

# 特定のバージョンをインストール
universal-asdf-plugin install <ツール> <バージョン>

# 最新の安定版を取得
universal-asdf-plugin latest-stable <ツール>

# ツールのヘルプを表示
universal-asdf-plugin help <ツール>

# .tool-versions を最新バージョンに更新
universal-asdf-plugin update-tool-versions
```

## 開発

### 前提条件

- Go 1.24+
- Docker（開発コンテナ用）

### 始め方

```bash
# リポジトリをクローン
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# VS Code で Dev Container を使用して開く
code universal-asdf-plugin.code-workspace

# またはローカルで実行
go build .
go test ./...
```

## ライセンス

Copyright 2025 Sumicare

このプロジェクトを使用することにより、Sumicare OSS [利用規約](./OSS_TERMS.JA.md)に同意したものとみなされます。

[Apache License, Version 2.0](../LICENSE) の下でライセンスされています。
