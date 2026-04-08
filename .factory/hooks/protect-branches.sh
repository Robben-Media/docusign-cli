#!/bin/bash
# Branch protection hook
# Prevents commits directly to protected branches.
# Install: .factory/hooks/protect-branches.sh
# Wire to PreToolUse matcher "Bash" in .factory/settings.json

set -e

input=$(cat)
tool_name=$(echo "$input" | jq -r '.tool_name')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

if [ "$tool_name" != "Bash" ]; then
  exit 0
fi

if ! echo "$command" | grep -qE "^git (commit|push)"; then
  exit 0
fi

cwd=$(echo "$input" | jq -r '.cwd')
cd "$cwd" 2>/dev/null || exit 0

if [ ! -d ".git" ]; then
  exit 0
fi

current_branch=$(git branch --show-current 2>/dev/null || echo "")

# Read protected branches from config or use defaults
protected_branches="main master production prod"

for branch in $protected_branches; do
  if [ "$current_branch" = "$branch" ]; then
    echo "Cannot commit directly to protected branch: $branch" >&2
    echo "Create a feature branch instead: git checkout -b feature/your-feature-name" >&2
    exit 2
  fi
done

exit 0
