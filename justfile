# project justfile

import? '.just/compliance.just'
import? '.just/gh-process.just'
import? '.just/pr-hook.just'
import? '.just/shellcheck.just'
import? '.just/cue-verify.just'
import? '.just/claude.just'
import? '.just/copilot.just'
import? '.just/repo-toml.just'
import? '.just/template-sync.just'

# list recipes (default works without naming it)
[group('Utility')]
list:
    just --list
    @echo "{{GREEN}}Your justfile is waiting for more scripts and snippets{{NORMAL}}"

# build the ctm binary
[group('Utility')]
build:
    go build -o ctm .

# run go tests
[group('Testing/Automation')]
test:
    go test ./...

# run go tests
[group('Testing/Automation')]
local-test: build
	sudo ./ctm status 192.168.42.208
	sudo ./ctm status 192.168.42.204

# release build platforms (goos-goarch pairs matching gh-observer release.yml)
release_platforms := "darwin-amd64 darwin-arm64 freebsd-386 freebsd-amd64 freebsd-arm64 linux-386 linux-amd64 linux-arm linux-arm64 windows-386 windows-amd64 windows-arm64"

# build cross-platform release binaries into dist/ as ctm-<goos>-<goarch>[.exe]
[group('Release')]
release-build:
    #!/usr/bin/env bash
    set -euo pipefail

    rm -rf dist/ctm-*
    # shellcheck disable=SC2043,SC1083  # {{release_platforms}} is a just variable, expanded before shellcheck sees it
    for platform in {{release_platforms}}; do
        goos="${platform%-*}"
        goarch="${platform#*-}"
        ext=""
        if [[ "$goos" == "windows" ]]; then
            ext=".exe"
        fi
        echo "{{BLUE}}Building ctm-${platform}${ext}...{{NORMAL}}"
        CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
            go build -trimpath -ldflags="-s -w" -o "dist/ctm-${platform}${ext}" .
    done
    ls -l dist/

# print base64 sha256 hashes of all release binaries (SLSA provenance subjects)
[group('Release')]
release-hashes:
    #!/usr/bin/env bash
    set -euo pipefail

    cd dist
    sha256sum ctm-* | base64 | tr -d '\n'

# sign release binaries with cosign (keyless, requires CI OIDC); run release-build first
[group('Release')]
release-sign:
    #!/usr/bin/env bash
    set -euo pipefail

    if ! command -v cosign >/dev/null 2>&1; then
        echo "{{RED}}Error: cosign not found on PATH{{NORMAL}}" >&2
        exit 1
    fi
    cd dist
    shopt -s nullglob
    binaries=(ctm-*)
    if [[ "${#binaries[@]}" -eq 0 ]]; then
        echo "{{RED}}Error: no ctm-* binaries in dist/, run 'just release-build' first{{NORMAL}}" >&2
        exit 1
    fi
    for binary in "${binaries[@]}"; do
        cosign sign-blob --yes "$binary" \
            --output-signature "${binary}.sig" \
            --output-certificate "${binary}.pem" \
            --bundle "${binary}.bundle"
    done
    ls -l ctm-*.sig ctm-*.pem ctm-*.bundle

# upload binaries and signatures to an existing GitHub release via gh release upload
[group('Release')]
release-upload rel_version:
    #!/usr/bin/env bash
    set -euo pipefail

    # shellcheck disable=SC2050  # rel_version is a variable
    if [[ ! "{{rel_version}}" =~ ^v[0-9]+(\.[0-9]+){0,2}$ ]]; then
        echo "{{RED}}Error: Release version must be vN[.N[.N]] (e.g. v1, v0.2, v2.3.4){{NORMAL}}" >&2
        exit 1
    fi
    cd dist
    shopt -s nullglob
    assets=(ctm-*)
    if [[ "${#assets[@]}" -eq 0 ]]; then
        echo "{{RED}}Error: no ctm-* assets in dist/, run 'just release-build' first{{NORMAL}}" >&2
        exit 1
    fi
    gh release upload "{{rel_version}}" --clobber -- "${assets[@]}"

# append binary verification instructions to an existing GitHub release's notes
[group('Release')]
release-notes rel_version:
    #!/usr/bin/env bash
    set -euo pipefail

    # shellcheck disable=SC2050  # rel_version is a variable
    if [[ ! "{{rel_version}}" =~ ^v[0-9]+(\.[0-9]+){0,2}$ ]]; then
        echo "{{RED}}Error: Release version must be vN[.N[.N]] (e.g. v1, v0.2, v2.3.4){{NORMAL}}" >&2
        exit 1
    fi
    BODY_FILE="$(mktemp)"
    trap 'rm -f "$BODY_FILE"' EXIT
    gh release view "{{rel_version}}" --json body --jq .body > "$BODY_FILE"

    # indentation is intentional: just dedents recipe bodies, so this EOF
    # lines up at column 0 for bash
    cat >> "$BODY_FILE" <<'EOF'

    ## Verifying the binaries

    Each `ctm-<os>-<arch>` binary is signed keyless with cosign (Sigstore).
    Verify a binary against its bundle:

        cosign verify-blob \
            --bundle ctm-<os>-<arch>.bundle \
            --certificate-oidc-issuer https://token.actions.githubusercontent.com \
            --certificate-identity-regexp '^https://github\.com/chicks-net/ctm/\.github/workflows/release\.yml@.+$' \
            ctm-<os>-<arch>

    Or check it the old-school way:

        cosign verify-blob --bundle ctm-<os>-<arch>.bundle ctm-<os>-<arch>

    SLSA-3 build provenance is attached to the release as `multiple.intoto.jsonl`.
    EOF

    gh release edit "{{rel_version}}" --notes-file "$BODY_FILE"

# post-release smoke test: confirm all expected assets landed on the release
[group('Release')]
release-check rel_version:
    #!/usr/bin/env bash
    set -euo pipefail

    # shellcheck disable=SC2050  # rel_version is a variable
    if [[ ! "{{rel_version}}" =~ ^v[0-9]+(\.[0-9]+){0,2}$ ]]; then
        echo "{{RED}}Error: Release version must be vN[.N[.N]] (e.g. v1, v0.2, v2.3.4){{NORMAL}}" >&2
        exit 1
    fi
    EXPECTED_ASSETS=0
    # shellcheck disable=SC2043,SC1083,SC2034  # {{release_platforms}} is a just variable, expanded before shellcheck sees it
    for platform in {{release_platforms}}; do
        EXPECTED_ASSETS=$((EXPECTED_ASSETS + 4))  # binary, .sig, .pem, .bundle
    done
    EXPECTED_ASSETS=$((EXPECTED_ASSETS + 1))  # multiple.intoto.jsonl from the provenance job
    ACTUAL_ASSETS=$(gh release view "{{rel_version}}" --json assets --jq '.assets | length')
    if [[ "$ACTUAL_ASSETS" -eq "$EXPECTED_ASSETS" ]]; then
        echo "{{GREEN}}Release {{rel_version}} has all $EXPECTED_ASSETS expected assets.{{NORMAL}}"
        exit 0
    fi
    echo "{{RED}}Release {{rel_version}} has $ACTUAL_ASSETS assets, expected $EXPECTED_ASSETS.{{NORMAL}}" >&2
    gh release view "{{rel_version}}" --json assets --jq '.assets[].name'
    exit 1

# report files that need gofmt
[group('Testing/Automation')]
fmt:
    @gofmt -l .
