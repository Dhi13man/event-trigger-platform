#!/bin/bash
set -e

echo "=== Fixing Branch Structure with Common History ==="
echo ""

# Get repo info
REPO_ROOT=$(git rev-parse --show-toplevel)
PDF_TEMP="/tmp/backend-assignment-oct-2025.pdf"

echo "Step 1: Saving PDF and current work..."
git checkout bootstrap-etp-golang
cp "$REPO_ROOT/docs/backend-assignment-oct-2025.pdf" "$PDF_TEMP"

# Get all the work commits (excluding the old base)
WORK_COMMITS=$(git log --oneline origin/main..bootstrap-etp-golang --reverse | awk '{print $1}')

echo ""
echo "Step 2: Creating new main branch with only PDF..."
git checkout --orphan main-new
git rm -rf . >/dev/null 2>&1 || true
mkdir -p docs
cp "$PDF_TEMP" docs/backend-assignment-oct-2025.pdf
git add docs/backend-assignment-oct-2025.pdf
git commit -m "docs: add backend assignment PDF

Initial commit with assignment requirements"

# Save the main commit hash
MAIN_COMMIT=$(git rev-parse HEAD)

echo ""
echo "Step 3: Creating new bootstrap branch based on main..."
git checkout -b bootstrap-new

# Cherry-pick all work commits or create a fresh commit with all files
echo ""
echo "Step 4: Adding all implementation work..."
git checkout bootstrap-etp-golang -- .
git reset HEAD docs/backend-assignment-oct-2025.pdf 2>/dev/null || true
git add .
git commit -m "feat: bootstrap event-trigger-platform with complete implementation

- API server with health check endpoint
- Scheduler service with 5-second ticker
- Worker consumer stub (user-implemented external service)
- Docker Compose setup (MySQL 8.0 + Kafka 3.8.1 KRaft mode)
- Database migrations (triggers, event_logs, idempotency_keys, retention)
- Multi-stage Dockerfile for optimized builds
- Configuration loading from environment variables
- Project structure: cmd/, internal/, platform/, pkg/
- MySQL Event Scheduler for automatic retention management
- Start scripts for easy setup

Implements bootstrap phase of event-trigger-platform per CLAUDE.md spec.
Platform publishes trigger events to Kafka; consumers are user-implemented."

echo ""
echo "Step 5: Replacing old branches..."
git checkout main-new
git branch -D main 2>/dev/null || true
git branch -m main

git checkout bootstrap-new
git branch -D bootstrap-etp-golang
git branch -m bootstrap-etp-golang

# Clean up temp
rm -f "$PDF_TEMP"

echo ""
echo "=== ✓ Branch Structure Fixed! ==="
echo ""
echo "Branch list:"
git branch -v
echo ""
echo "Commit graph:"
git log --oneline --all --graph --decorate -5
echo ""
echo "Main branch contents:"
git ls-tree -r main --name-only
echo ""
echo "Bootstrap branch file count:"
git ls-tree -r bootstrap-etp-golang --name-only | wc -l
echo ""
echo "=== Next Steps ==="
echo "1. Force push both branches: git push -f origin main bootstrap-etp-golang"
echo "2. Create PR from bootstrap-etp-golang to main (they now share history!)"
