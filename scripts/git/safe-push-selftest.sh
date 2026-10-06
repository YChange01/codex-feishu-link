#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/safe-push-selftest.XXXXXX")"
trap 'rm -rf "${TEST_ROOT}"' EXIT
export GIT_TERMINAL_PROMPT=0
export SAFE_PUSH_GUARD_LOG="${TEST_ROOT}/guards.log"
export SAFE_PUSH_TEST_MARKER="${TEST_ROOT}/tested"

fail() { echo "FAIL: $*" >&2; cat "${TEST_ROOT}/output.log" >&2; exit 1; }

configure_repo() {
  git -C "$1" config user.name 'Safe Push Test'
  git -C "$1" config user.email 'safe-push@example.invalid'
  git -C "$1" config commit.gpgsign false
  git -C "$1" config core.hooksPath /dev/null
}

new_fixture() {
  local name="$1"
  CLIENT="${TEST_ROOT}/${name}/client"
  REMOTE_REPO="${TEST_ROOT}/${name}/remote.git"
  mkdir -p "${CLIENT}/scripts/git" "${CLIENT}/scripts/check"
  git init -q --bare -b main "${REMOTE_REPO}"
  git init -q -b main "${CLIENT}"
  configure_repo "${CLIENT}"
  cp "${SCRIPT_DIR}/safe-push.sh" "${CLIENT}/scripts/git/safe-push.sh"
  local guard
  for guard in go-format no-local-paths no-legacy-names feishu-call-broker; do
    cat > "${CLIENT}/scripts/check/${guard}.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "${0##*/}" >> "${SAFE_PUSH_GUARD_LOG}"
[[ "${SAFE_PUSH_FAIL_GUARD:-}" != "${0##*/}" ]]
EOF
  done
  git -C "${CLIENT}" add .
  git -C "${CLIENT}" commit -qm fixture
  git -C "${CLIENT}" remote add origin "${REMOTE_REPO}"
  : > "${SAFE_PUSH_GUARD_LOG}"
  rm -f "${SAFE_PUSH_TEST_MARKER}"
}

run_helper() {
  bash "${CLIENT}/scripts/git/safe-push.sh" "$@" > "${TEST_ROOT}/output.log" 2>&1
}

expect_failure() {
  local expected="$1" actual
  shift
  if run_helper "$@"; then
    fail "expected exit ${expected}, got success"
  else
    actual=$?
    [[ "${actual}" == "${expected}" ]] || fail "expected exit ${expected}, got ${actual}"
  fi
}

assert_remote_empty() {
  [[ -z "$(git --git-dir="${REMOTE_REPO}" for-each-ref refs/heads/)" ]] || fail 'remote changed unexpectedly'
}

advance_both_branches() {
  run_helper --no-test || fail 'initial fixture push'
  local peer="${CLIENT%/client}/peer"
  git clone -q "${REMOTE_REPO}" "${peer}"
  configure_repo "${peer}"
  echo upstream > "${peer}/upstream.txt"
  git -C "${peer}" add upstream.txt
  git -C "${peer}" commit -qm upstream
  git -C "${peer}" push -q origin main
  UPSTREAM_HEAD="$(git -C "${peer}" rev-parse HEAD)"
  echo local > "${CLIENT}/local.txt"
  git -C "${CLIENT}" add local.txt
  git -C "${CLIENT}" commit -qm local
}

new_fixture first
run_helper --always-test --test-cmd 'printf tested > "$SAFE_PUSH_TEST_MARKER"' || fail 'first push'
[[ -f "${SAFE_PUSH_TEST_MARKER}" ]] || fail '--always-test was skipped'
[[ "$(wc -l < "${SAFE_PUSH_GUARD_LOG}" | tr -d ' ')" == 4 ]] || fail 'guardrails were skipped'
[[ "$(git --git-dir="${REMOTE_REPO}" rev-parse refs/heads/main)" == "$(git -C "${CLIENT}" rev-parse HEAD)" ]] || fail 'first push head mismatch'
echo 'PASS: first push preserves guardrails and --always-test'

run_helper --test-cmd false || fail 'existing branch without rebase'
echo 'PASS: existing branch without rebase keeps default test behavior'

new_fixture rebase
advance_both_branches
run_helper --confirm-rebase-review --test-cmd 'printf tested > "$SAFE_PUSH_TEST_MARKER"' || fail 'reviewed rebase'
[[ -f "${SAFE_PUSH_TEST_MARKER}" ]] || fail 'post-rebase tests were skipped'
git -C "${CLIENT}" merge-base --is-ancestor "${UPSTREAM_HEAD}" HEAD || fail 'remote commit lost'
[[ -f "${CLIENT}/local.txt" && -f "${CLIENT}/upstream.txt" ]] || fail 'rebase content lost'
[[ "$(git --git-dir="${REMOTE_REPO}" rev-parse main)" == "$(git -C "${CLIENT}" rev-parse HEAD)" ]] || fail 'rebased push head mismatch'
echo 'PASS: existing divergent branch rebases, tests, and pushes after review'

new_fixture review
advance_both_branches
expect_failure 4 --test-cmd true < /dev/null
[[ "$(git --git-dir="${REMOTE_REPO}" rev-parse main)" == "${UPSTREAM_HEAD}" ]] || fail 'unreviewed rebase was pushed'
echo 'PASS: post-rebase review remains required'

new_fixture unavailable
git -C "${CLIENT}" remote set-url origin "${TEST_ROOT}/does-not-exist.git"
expect_failure 128 --no-test
if grep -q 'skip rebase for first push' "${TEST_ROOT}/output.log"; then fail 'transport error treated as missing branch'; fi
assert_remote_empty
echo 'PASS: remote access failure aborts instead of permitting first push'

new_fixture guard
export SAFE_PUSH_FAIL_GUARD=no-local-paths.sh
expect_failure 1 --no-test
unset SAFE_PUSH_FAIL_GUARD
assert_remote_empty
echo 'PASS: guardrail failure prevents first push'

new_fixture tests
expect_failure 3 --always-test --test-cmd false
assert_remote_empty
echo 'PASS: test failure prevents first push'

new_fixture dirty
echo untracked > "${CLIENT}/untracked.txt"
expect_failure 1 --no-test
assert_remote_empty
echo 'PASS: dirty worktree prevents first push'
