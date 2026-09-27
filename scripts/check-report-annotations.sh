#!/usr/bin/env bash
# Verifies that archived status/planning reports carry inline resolution
# markers: every file in docs/status/archived/ and docs/planning/archived/
# must contain at least one ~~strikethrough~~ annotation.
#
# The marker conventions come from the docs-health skill:
#   ~~...~~ done (docs-health pass ...)   — verified resolved
#   ~~...~~ not-do (reason)               — explicitly rejected
#
# Usage: scripts/check-report-annotations.sh   (run from the repo root)

set -euo pipefail

fail=0

for dir in docs/status/archived docs/planning/archived; do
	[ -d "$dir" ] || continue
	while IFS= read -r -d '' file; do
		if ! grep -q '~~' "$file"; then
			echo "MISSING markers: $file"
			fail=1
		fi
	done < <(find "$dir" -name '*.md' ! -name 'README.md' -print0)
done

if [ "$fail" -eq 0 ]; then
	echo "OK: all archived reports carry inline resolution markers"
fi

exit "$fail"
