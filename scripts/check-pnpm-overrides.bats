#!/usr/bin/env bats
# shellcheck disable=SC2154  # bats sets BATS_TEST_DIRNAME and BATS_TEST_TMPDIR

setup() {
  script="$BATS_TEST_DIRNAME/check-pnpm-overrides.sh"
  workspace="$BATS_TEST_TMPDIR/pnpm-workspace.yaml"
}

@test "passes when every override has a commented advisory" {
  cat > "$workspace" << 'EOF'
allowBuilds:
  esbuild: true

overrides:
  # postcss 8.4.31 (GHSA-qx2v-qp2m-jg93); 8.5.10 fixes it.
  postcss@<8.5.10: ^8.5.10
  # Fixes CVE-2026-12345.
  sharp@<0.35.5: ^0.35.5
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 0 ]]
}

@test "accepts a multi-line comment naming several advisories" {
  cat > "$workspace" << 'EOF'
overrides:
  # brace-expansion is vulnerable to DoS
  # (GHSA-6j4f-fj2g-mc7p, GHSA-qhr7-859c-m2p7).
  brace-expansion@<5.0.12: ^5.0.12
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 0 ]]
}

@test "fails on an override with no comment" {
  cat > "$workspace" << 'EOF'
overrides:
  # postcss (GHSA-qx2v-qp2m-jg93)
  postcss@<8.5.10: ^8.5.10
  fast-uri@<3.1.8: ^3.1.8
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"$workspace:4: override 'fast-uri@<3.1.8'"* ]]
  [[ "$output" != *postcss* ]]
}

@test "fails when the comment names no advisory ID" {
  cat > "$workspace" << 'EOF'
overrides:
  # Security fix.
  sharp@<0.35.5: ^0.35.5
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"override 'sharp@<0.35.5'"* ]]
}

@test "does not carry a comment across a blank line" {
  cat > "$workspace" << 'EOF'
overrides:
  # postcss (GHSA-qx2v-qp2m-jg93)

  postcss@<8.5.10: ^8.5.10
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 1 ]]
}

@test "does not reuse one entry's comment for the next entry" {
  cat > "$workspace" << 'EOF'
overrides:
  # postcss (GHSA-qx2v-qp2m-jg93)
  postcss@<8.5.10: ^8.5.10
  sharp@<0.35.5: ^0.35.5
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"override 'sharp@<0.35.5'"* ]]
}

@test "ignores keys outside overrides" {
  cat > "$workspace" << 'EOF'
overrides:
  # GHSA-qx2v-qp2m-jg93
  postcss@<8.5.10: ^8.5.10
allowBuilds:
  esbuild: true
EOF
  run "$script" "$workspace"
  [[ "$status" -eq 0 ]]
}

@test "fails with a usage message when given no files" {
  run "$script"
  [[ "$status" -eq 2 ]]
  [[ "$output" == *usage* ]]
}

@test "fails when a file is missing" {
  run "$script" "$BATS_TEST_TMPDIR/absent.yaml"
  [[ "$status" -eq 2 ]]
  [[ "$output" == *"absent.yaml"* ]]
}
