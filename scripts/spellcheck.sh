#!/usr/bin/env bash
# Spellcheck using cspell with words from workspace config
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

trap 'rm -f .cspell-temp-config.json' EXIT

# Extract words array from workspace file
WORDS=$(sed -n '/"cSpell.words": \[/,/\]/p' universal-asdf-plugin.code-workspace 2>/dev/null | sed 's/"cSpell.words": //' | sed 's/],$/]/' || echo "[]")
[[ -z "$WORDS" ]] && WORDS="[]"

cat >.cspell-temp-config.json <<EOF
{
  "version": "0.2",
  "language": "en",
  "words": ${WORDS},
  "ignorePaths": [
    ".git/**", "*.lock", "go.mod", "go.sum","*.CS.md", "*.DE.md", "*.FR.md", "*.JA.md", "*.NO.md", "*.PL.md", "*.RO.md", "*.UA.md", "*.ZH.md",
    "go.work.sum", "*.min.*", "*.log",
    "**/coverage/**", "**/build/**", "**/mutation-reports/**"
  ],
  "ignoreRegExpList": ["/[А-Яа-яЁёІіЇїЄєҐґ]+/g", "/[\\\\u0400-\\\\u04FF]+/g"]
}
EOF

cspell --config=.cspell-temp-config.json --no-progress --show-context --cache --cache-location .cspell-cache "./**" "$@"
