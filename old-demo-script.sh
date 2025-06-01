#!/usr/bin/env bash
set -euo pipefail

# Simple CI-Feedback Loop for Amp CLI
# Usage: ./amp-agent-simple.sh --task "your task" --repo "git@github.com:org/repo.git"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

usage() {
    cat << EOF
Usage: $0 --task "TASK" --repo "REPO_URL"

Required:
  --task TASK       Natural language task description
  --repo REPO_URL   Git repository URL (SSH or HTTPS)

Examples:
  $0 --task "fix the power function to handle negative exponents correctly" --repo git@github.com:brettsmith212/ci-test.git
EOF
}

# Parse command line arguments
TASK=""
REPO_URL=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --task)
            TASK="$2"
            shift 2
            ;;
        --repo)
            REPO_URL="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$TASK" ]]; then
    log_error "Task description is required"
    usage
    exit 1
fi

if [[ -z "$REPO_URL" ]]; then
    log_error "Repository URL is required"
    usage
    exit 1
fi

# Check dependencies
for dep in git amp gh; do
    if ! command -v "$dep" &> /dev/null; then
        log_error "Required dependency not found: $dep"
        if [[ "$dep" == "gh" ]]; then
            log_error "GitHub CLI is required to monitor CI status. Install: https://github.com/cli/cli"
        fi
        exit 1
    fi
done

# Setup working directory
WORK_DIR="/tmp/amp-agent-$(date +%s)"
mkdir -p "$WORK_DIR"

# Cleanup on exit
cleanup() {
    if [[ -d "$WORK_DIR" ]]; then
        log_info "Cleaning up working directory: $WORK_DIR"
        rm -rf "$WORK_DIR"
    fi
}
trap cleanup EXIT

log_info "Starting Amp CI-Feedback Loop Agent"
log_info "Task: $TASK"
log_info "Repo: $REPO_URL"
log_info "Working directory: $WORK_DIR"

# Clone repository
cd "$WORK_DIR"
log_info "Cloning repository..."
git clone "$REPO_URL" repo
cd repo

# Generate comprehensive prompt for Amp
PROMPT="You are a coding agent working on a software project. Your task is to implement changes and ensure they pass CI before pushing to origin.

TASK: $TASK

WORKFLOW TO FOLLOW:
1. Create a new feature branch (name it something like \"amp/fix-TIMESTAMP\")
2. Implement the requested changes to accomplish the task
3. Commit your changes with a meaningful commit message
4. Push your branch to origin to trigger GitHub Actions CI
5. Monitor the GitHub Actions workflow status using: 'gh run list --branch BRANCH_NAME --limit 1'
6. Wait for CI completion using: 'gh run watch RUN_ID' or check status periodically
7. If CI fails, get detailed logs using: 'gh run view RUN_ID --log'
8. Analyze the failure logs, make fixes, commit, and push again to re-trigger CI
9. Repeat steps 5-8 until all GitHub Actions checks pass
10. Report completion with the branch name and final commit SHA

IMPORTANT GUIDELINES:
- Look at the existing code structure and follow the same patterns
- Make minimal, focused changes
- Ensure your solution handles edge cases properly
- Use GitHub Actions CI as your testing mechanism - don't run tests locally
- Push early and often to get CI feedback
- Read CI logs carefully to understand failures
- Be thorough in analyzing and fixing CI failures

Please start by examining the repository structure and understanding the codebase, then proceed with the workflow above."

log_info "Sending task to Amp..."
log_info "Waiting for Amp to complete the workflow..."

# Execute Amp with the prompt
if echo "$PROMPT" | amp; then
    log_success "Amp completed the task successfully!"
    log_info "Check the repository for the new feature branch with your changes."
else
    log_error "Amp failed to complete the task"
    exit 1
fi
