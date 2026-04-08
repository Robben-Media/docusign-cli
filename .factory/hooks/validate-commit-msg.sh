#!/bin/bash
# Git commit message validator
# Enforces conventional commits and co-authorship trailer.
# Install: .factory/hooks/validate-commit-msg.sh
# Wire to PreToolUse matcher "Bash" in .factory/settings.json

set -e

input=$(cat)
tool_name=$(echo "$input" | jq -r '.tool_name')

if [ "$tool_name" != "Bash" ]; then
  exit 0
fi

command=$(echo "$input" | jq -r '.tool_input.command')

if ! echo "$command" | grep -qE "^git commit"; then
  exit 0
fi

# Extract commit message from -m flag
if ! echo "$command" | grep -qE "git commit.*-m"; then
  exit 0
fi

commit_msg=$(echo "$command" | sed -E 's/.*git commit.*-m[= ]*["\x27]([^"\x27]+)["\x27].*/\1/')

# Validate conventional commit format
if ! echo "$commit_msg" | grep -qE "^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?:.+"; then
  echo "Invalid commit message format" >&2
  echo "" >&2
  echo "Must follow: type(scope): description" >&2
  echo "Valid types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert" >&2
  echo "Example: feat(auth): add user login" >&2
  exit 2
fi

# Warn if no co-authorship trailer
if ! echo "$commit_msg" | grep -qE "Co-authored-by:"; then
  echo "Note: No Co-authored-by trailer found. Droid should add this automatically." >&2
fi

exit 0
