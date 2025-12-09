#!/usr/bin/env bash
# Add unknown words from spellcheck to workspace config
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

WORKSPACE="universal-asdf-plugin.code-workspace"

# Get unknown words from spellcheck
new_words=$(./scripts/spellcheck.sh 2>&1 | sed -nE 's/.*Unknown word \(([a-zA-Z0-9._-]+)\).*/\L\1/p' | sort -u || true)
[[ -z "$new_words" ]] && { echo "No new words to add"; exit 0; }

echo "Adding: $(echo "$new_words" | tr '\n' ' ')"

# Extract existing words from cSpell.words section only
existing_words=$(sed -n '/"cSpell.words": \[/,/\]/p' "$WORKSPACE" | grep -oE '"[a-zA-Z0-9._-]+"' | tr -d '"' || true)

# Merge and sort all words
all_words=$(printf '%s\n%s' "$existing_words" "$new_words" | grep -v '^$' | sort -u)

# Use jq to update the file (preserves formatting)
jq --argjson words "$(echo "$all_words" | jq -R . | jq -s .)" \
  '.settings["cSpell.words"] = ($words | sort)' "$WORKSPACE" > "${WORKSPACE}.tmp"
mv "${WORKSPACE}.tmp" "$WORKSPACE"

echo "Done - $(echo "$all_words" | wc -l) words total"
