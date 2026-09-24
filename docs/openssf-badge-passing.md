# Filling Out the OpenSSF Best Practices Badge

This is a copy-paste companion for working through the badge form at
<https://www.bestpractices.dev/> once the `chicks-net/ctm` project page
exists — sign up with the GitHub account that owns the project, add
project `chicks-net/ctm`, and the page number this doc should be updated
with appears in the URL.  (Everywhere below says "project page"; after
signup, read "the edit form on your project page".)

The goal here is the classic **passing** badge.  The project page also
shows "Baseline Level 1/2/3" meters; that's a separate, newer OpenSSF
Baseline initiative and out of scope for this document.  (Getting to
silver is a follow-on exercise; this repo's gap analysis lives in
[#48](https://github.com/chicks-net/ctm/issues/48).)

With this doc open in one window and the form in another, the whole thing
should take under an hour.

## How the form works

- Log in at <https://www.bestpractices.dev/> using the GitHub account that
  owns the project (`chicks-net`), open the project page, and click
  **Edit**.
- The passing tier has 67 criteria in six sections: Basics (13), Change
  Control (9), Reporting (8), Quality (13), Security (16), Analysis (8).
  The section counters (`0/13`, etc.) tell you what's left.
- Each criterion takes one of four answers: **Met**, **Unmet**, **N/A**
  (where offered), or **Future**. To earn the passing badge: all MUST and
  MUST NOT criteria Met, all SHOULD criteria Met *or* Unmet with a
  justification, and all SUGGESTED criteria at least answered (Met, Unmet,
  or Future).
- Many criteria have a URL field — paste the evidence link exactly where
  this doc says `(URL)`.  Use `blob/main/...` links so they stay stable.
- Save after each section.  The form is long and you do not want to find out
  what its session timeout feels like the hard way.
- If you enter justification text that is a generic comment rather than a
  rationale, start it with `//` and a space.

## Quick facts for the General section

These are the short-answer fields at the top of the Basics section, before
the scored criteria.

| Field | Answer |
| --- | --- |
| Human-readable name | `ctm` |
| Brief description | Keep the existing one (control Time Machines Corp. clocks over UDP — status, timers, colors, dimming) |
| Project URL | `https://github.com/chicks-net/ctm` |
| VCS repository URL | `https://github.com/chicks-net/ctm` (same thing, and that's fine) |
| License | `GPL-2.0-only` (pick it from the SPDX dropdown) |
| Implementation languages | `Go` |
| CPE name | Leave blank — this project has no CPE entry |

## Basics (13 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 1 | `description_good` (MUST) | Met | The README opens with a succinct description of what the tool does — control Time Machines Corp. network clocks over UDP — plus a status line on exactly which clock APIs are supported. | `blob/main/README.md` |
| 2 | `interact` (MUST) | Met | The README documents how to obtain the software (Homebrew, binary from releases, build from source), how to provide feedback (GitHub issues, SECURITY.md for security issues), and how to contribute (CONTRIBUTING.md). | `blob/main/README.md` |
| 3 | `contribution` (MUST) (URL) | Met | The project uses GitHub pull requests and an issue tracker; CONTRIBUTING.md walks through the whole contribution cycle (`just branch` → commits → `just pr` → `just merge`). | `blob/main/.github/CONTRIBUTING.md` |
| 4 | `contribution_requirements` (SHOULD) (URL) | Met | CONTRIBUTING.md sets the requirements for acceptable contributions: discuss major changes in an issue first, good commit messages, check CI on your PR; pre-commit hooks (gitleaks, shellcheck, whitespace hygiene) and CI lint/format gates enforce style mechanically. | `blob/main/.github/CONTRIBUTING.md` |
| 5 | `floss_license` (MUST) | Met | Released under GPL-2.0-only, an OSI-approved and FSF-free license. Full text is in the top-level LICENSE file. | `blob/main/LICENSE` |
| 6 | `floss_license_osi` (SUGGESTED) | Met | GPL-2.0 is approved by the Open Source Initiative. | `blob/main/LICENSE` |
| 7 | `license_location` (MUST) (URL) | Met | The license is posted as a top-level file named LICENSE, the standard location. | `blob/main/LICENSE` |
| 8 | `documentation_basics` (MUST) | Met | The README covers installation (Homebrew, release binaries, build from source) and full usage of every subcommand with flags, examples, and sample output; SECURITY.md covers secure use and the protocol's real security limitations. | `blob/main/README.md` |
| 9 | `documentation_interface` (MUST) | Met | The external interface is a CLI. The README documents the invocation pattern, global flags, all 16 subcommands with per-command argument syntax, and what status output looks like; `ctm help` and `ctm $SUBCOMMAND -h` print the same information generated from the flag definitions. | `blob/main/README.md` |
| 10 | `sites_https` (MUST) | Met | The project website, repository, and download URLs are all GitHub HTTPS URLs. | `https://github.com/chicks-net/ctm` |
| 11 | `discussion` (MUST) | Met | GitHub issue and pull request discussions: searchable, URL-addressable, open to new participants, no proprietary client required. | `https://github.com/chicks-net/ctm/issues` |
| 12 | `english` (SHOULD) | Met | All documentation, code, and issue discussions are in English. | `blob/main/README.md` |
| 13 | `maintained` (MUST) | Met | The project is actively maintained: recent tagged releases (v0.1, v0.2 within days of each other in September 2026), frequent commits, and issues are triaged and closed. | `https://github.com/chicks-net/ctm/releases` |

## Change Control (9 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 14 | `repo_public` (MUST) | Met | The source repository is publicly readable at the project URL. | `https://github.com/chicks-net/ctm` |
| 15 | `repo_track` (MUST) | Met | Git tracks what changed, who changed it, and when for every commit. | `https://github.com/chicks-net/ctm/commits/main/` |
| 16 | `repo_interim` (MUST) | Met | Development happens on `main` with per-PR branches; all interim versions between releases are public in the repository. | `https://github.com/chicks-net/ctm/commits/main/` |
| 17 | `repo_distributed` (SUGGESTED) | Met | The project uses git. | `https://github.com/chicks-net/ctm` |
| 18 | `version_unique` (MUST) | Met | Every release has a unique version identifier: a git tag (v0.1, v0.2). | `https://github.com/chicks-net/ctm/tags` |
| 19 | `version_semver` (SUGGESTED) | Met | Tags use `vMAJOR.MINOR` semantic-versioning style. | `https://github.com/chicks-net/ctm/tags` |
| 20 | `version_tags` (SUGGESTED) | Met | Each release is identified in the version control system with a git tag. | `https://github.com/chicks-net/ctm/tags` |
| 21 | `release_notes` (MUST) (URL) | Met | Every release on the GitHub releases page has human-readable release notes summarizing merged pull requests. | `https://github.com/chicks-net/ctm/releases` |
| 22 | `release_notes_vulns` (MUST, N/A) | N/A | No publicly known runtime vulnerabilities with CVE assignments have been fixed in any release — no CVEs exist for this project. Revisit if one is ever assigned. | `https://github.com/chicks-net/ctm/releases` |

## Reporting (8 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 23 | `report_process` (SHOULD) (URL) | Met | Users submit bug reports via the GitHub issue tracker, which provides templates (feature request and blank checklists); CONTRIBUTING.md's "Reporting problems" section tells users what a good report includes (exact error messages, text not screenshots). | `https://github.com/chicks-net/ctm/issues` |
| 24 | `report_tracker` (SHOULD) | Met | GitHub's issue tracker is used for individual issues. | `https://github.com/chicks-net/ctm/issues` |
| 25 | `report_responses` (MUST) | Met | The maintainer acknowledges and triages the majority of bug reports; responses need not include fixes. | `https://github.com/chicks-net/ctm/issues?q=is%3Aissue` |
| 26 | `enhancement_responses` (SHOULD) | Met | The maintainer responds to enhancement requests, including requests that get declined with rationale. | `https://github.com/chicks-net/ctm/issues?q=is%3Aissue+label%3Aenhancement` |
| 27 | `report_archive` (MUST) (URL) | Met | Issues and pull requests are publicly archived and searchable. | `https://github.com/chicks-net/ctm/issues?q=` |
| 28 | `vulnerability_report_process` (MUST) (URL) | Met | SECURITY.md publishes the process for reporting vulnerabilities, including confidential disclosure via email with an expected format. | `blob/main/.github/SECURITY.md` |
| 29 | `vulnerability_report_private` (MUST) (URL) | Met | SECURITY.md's Confidential Disclosure section provides a private email channel for vulnerability reports. | `blob/main/.github/SECURITY.md` |
| 30 | `vulnerability_report_response` (MUST, N/A) | N/A | No vulnerability reports have been received in the last 6 months. | (none) |

## Quality (13 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 31 | `build` (MUST) | Met | `just build` (wrapping `go build -o ctm .`) rebuilds the binary from modified source; documented in the README, with `make` as an alternative. | `blob/main/README.md` |
| 32 | `build_common_tools` (SUGGESTED) | Met | Built with the standard Go toolchain; task automation via `just`, a widely used FLOSS command runner, with a Makefile fallback. | `blob/main/justfile` |
| 33 | `build_floss_tools` (SHOULD) | Met | The entire build toolchain — Go compiler, just, pre-commit — is FLOSS. | `blob/main/go.mod` |
| 34 | `test` (MUST) | Met | 40 test functions including 5 fuzz targets (time parsing, color spec, dimmer level, display modes, status decode), run with standard `go test ./...` and documented in CLAUDE.md; CI runs the suite on every push and pull request. Statement coverage is ~86%. | `blob/main/.github/workflows/go-ci.yml` |
| 35 | `test_invocation` (SHOULD) | Met | Standard Go invocation: `go test ./...` (wrapped as `just test`). | `blob/main/README.md` |
| 36 | `test_most` (SUGGESTED) | Met | Tests cover the wire-format structs, command bytes, time parsing, subcommand dispatch (driving the real CLI tree with a fake connection), and loopback UDP dialing, plus 5 fuzz targets over the security-relevant parsers. | `blob/main/main_test.go` |
| 37 | `test_continuous_integration` (SUGGESTED) | Met | The Go CI workflow runs format check, build, vet, and the full test suite on every push to main and every pull request. | `blob/main/.github/workflows/go-ci.yml` |
| 38 | `test_policy` (MUST) | Met | The PR template's verify section requires submitters to state how they verified the change, including positive and negative test cases; tests run as a blocking CI gate on every PR. | `blob/main/.github/pull_request_template.md` |
| 39 | `tests_are_added` (MUST) | Met | Recent features each shipped with new tests: `color_set` and `dimmer_set` added parser and dispatch tests alongside the command implementations. | `blob/main/main_test.go` |
| 40 | `tests_documented_added` (SUGGESTED) | Met | The PR template documents the expectation that changes come with verification (including test cases); CI enforces it by running the suite on every PR. | `blob/main/.github/pull_request_template.md` |
| 41 | `warnings` (MUST) | Met | CI gates on `go vet` and `gofmt -l` (build fails if any file needs formatting); the Go compiler itself fails on a broad class of common mistakes by default. Pre-commit hooks catch issues before they reach CI. | `blob/main/.github/workflows/go-ci.yml` |
| 42 | `warnings_fixed` (MUST) | Met | Format and vet findings block the CI gate; the codebase is kept warning-clean. | `blob/main/.github/workflows/go-ci.yml` |
| 43 | `warnings_strict` (SUGGESTED) | Met | Go's compiler is strict by default (unused variables/imports and type errors are fatal), supplemented by `go vet` and the gofmt gate on every push and PR. | `blob/main/.github/workflows/go-ci.yml` |

## Security (16 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 44 | `know_secure_design` (MUST) | Met | SECURITY.md documents the project's security posture honestly: the clock protocol is unauthenticated cleartext UDP, anyone on the network can send commands, and the tool is meant for trusted networks — that's the design analysis, written down. | `blob/main/.github/SECURITY.md` |
| 45 | `know_common_errors` (MUST) | Met | SECURITY.md's Security Risks section enumerates the common error classes for this domain: unauthenticated commands, cleartext traffic observation, and spoofed responses — with the practical mitigation. | `blob/main/.github/SECURITY.md` |
| 46 | `crypto_published` (MUST, N/A) | N/A | ctm performs no cryptographic operations at all — it exchanges fixed-format UDP packets with a clock and holds no secrets; standard published crypto is used only transitively in release signing (cosign/Sigstore) and HTTPS delivery via GitHub. | (none) |
| 47 | `crypto_call` (MUST, N/A) | N/A | The application itself calls no cryptographic primitives. Release signing uses cosign (Sigstore, keyless OIDC), a well-maintained FLOSS implementation. | `blob/main/.github/workflows/release.yml` |
| 48 | `crypto_floss` (MUST, N/A) | N/A | All cryptographic tooling in use (Sigstore/cosign for signing, GitHub TLS for delivery) is FLOSS; the application itself uses none. | `blob/main/.github/workflows/release.yml` |
| 49 | `crypto_keylength` (MUST, N/A) | N/A | ctm generates or manages no cryptographic keys; TLS uses GitHub's certificates and release signing uses Sigstore keyless OIDC (no project-held keys). | (none) |
| 50 | `crypto_working` (MUST, N/A) | N/A | The application performs no crypto that could work or not work; delivery security is GitHub TLS, whose correct operation is visible every time a release is downloaded. | (none) |
| 51 | `crypto_weaknesses` (MUST, N/A) | N/A | The application performs no cryptography, so it introduces no cryptographic weaknesses. | (none) |
| 52 | `crypto_pfs` (MUST, N/A) | N/A | The application performs no TLS of its own; downloads ride GitHub's HTTPS, which provides modern key exchange. | (none) |
| 53 | `crypto_password_storage` (MUST, N/A) | N/A | ctm stores no passwords or credentials of any kind — it has no config file, no auth, and no secrets; it sends fixed-format UDP packets. | `blob/main/README.md` |
| 54 | `crypto_random` (MUST, N/A) | N/A | The application uses no randomness; it is a deterministic UDP request/response tool. | (none) |
| 55 | `delivery_mitm` (MUST) | Met | The software is delivered via GitHub over HTTPS, which counters MITM attacks; binaries are additionally cosign-signed (keyless) and carry GitHub build attestations plus SLSA provenance. | `https://github.com/chicks-net/ctm/releases` |
| 56 | `delivery_unsigned` (MUST NOT) | Met | Binaries and signatures are only distributed over HTTPS on the GitHub releases page; binaries carry cosign keyless signatures, build attestations, and SLSA provenance. | `https://github.com/chicks-net/ctm/releases` |
| 57 | `vulnerabilities_fixed_60_days` (MUST) | Met | There are no publicly known vulnerabilities of medium or higher severity; the project has no CVEs and SECURITY.md commits to shipping security updates in new releases with a release-notes notification. | `blob/main/.github/SECURITY.md` |
| 58 | `vulnerabilities_critical_fixed` (SHOULD) | Met | No critical vulnerabilities are known; SECURITY.md's Security Updates section commits to including security fixes in releases. | `blob/main/.github/SECURITY.md` |
| 59 | `no_leaked_credentials` (MUST NOT) | Met | gitleaks runs in CI on every push and PR and as a pre-commit hook locally; no valid private credentials are committed. | `blob/main/.github/workflows/gitleaks.yml` |

## Analysis (8 criteria)

| # | Criterion (level) | Answer | Paste-ready justification | Evidence URL |
| --- | --- | --- | --- | --- |
| 60 | `static_analysis` (MUST) | Met | CodeQL performs static analysis of the Go code on every push and pull request; supplemented by go vet and the gofmt gate in CI, plus workflow-linting (actionlint, zizmor) and infrastructure scanning (checkov). | `blob/main/.github/workflows/codeql.yml` |
| 61 | `static_analysis_common_vulnerabilities` (SUGGESTED) | Met | CodeQL's Go security query pack checks for common vulnerability classes (CWE-mapped); results upload to GitHub code scanning. | `blob/main/.github/workflows/codeql.yml` |
| 62 | `static_analysis_fixed` (MUST) | Met | CodeQL findings surface as GitHub code scanning alerts and are triaged; no medium or higher severity exploitable vulnerabilities are open. | `https://github.com/chicks-net/ctm/security/code-scanning` |
| 63 | `static_analysis_often` (SUGGESTED) | Met | Static analysis runs on every push and every pull request. | `blob/main/.github/workflows/codeql.yml` |
| 64 | `dynamic_analysis` (SUGGESTED) | Met | Five native Go fuzz targets cover the security-relevant input parsing (colon-delimited time parts, hex color specs, dimmer levels, display mode names, raw status packet decoding); seed corpora run as ordinary tests in CI on every push and PR, and extended fuzzing is invocable via `go test -fuzz`. | `blob/main/main_test.go` |
| 65 | `dynamic_analysis_unsafe` (SUGGESTED, N/A) | N/A | The project is pure Go with no `unsafe` usage — memory-safe, no memory-unsafe languages. | `blob/main/go.mod` |
| 66 | `dynamic_analysis_enable_assertions` (SUGGESTED) | Met | Go test and fuzz builds enable extensive runtime checks (bounds checks, nil-dereference and type-assertion panics); fuzzing exercises the full parser surface with mutated inputs, and fuzz seed corpora run in CI. | `blob/main/main_test.go` |
| 67 | `dynamic_analysis_fixed` (MUST, N/A) | Met | Fuzzing has found no exploitable vulnerabilities; fuzz seed corpora run in CI on every push and PR so regressions are caught before release. | `blob/main/main_test.go` |

## Verify before you answer

These criteria make claims about recent activity that only you can
confirm.  Spend five minutes on each before clicking the radio button —
the badge is self-certified, so honesty is the whole product:

- **13 `maintained`** — is there a release and reasonable commit activity
  within roughly the last year? (At time of writing: yes, comfortably.)
- **25 `report_responses` / 26 `enhancement_responses`** — scan the issue
  list for the last 2-12 months.  Have most reports and requests gotten
  some reply, even if the reply was "no"?  If honestly no, mark Unmet
  with a justification — SHOULD criteria may be Unmet-with-justification
  without blocking the badge.
- **30 `vulnerability_report_response`** — N/A only holds if no one has
  reported a vulnerability in the last 6 months.  If someone did, the
  initial response must have been within 14 days.
- **57 `vulnerabilities_fixed_60_days`** — confirm there is no publicly
  known (e.g., NVD-listed) vulnerability older than 60 days.  With zero
  CVEs for this project, this should be quick.
- **62 `static_analysis_fixed`** — glance at the
  [code scanning alerts](https://github.com/chicks-net/ctm/security/code-scanning)
  page and make sure it's actually clean before pasting that
  justification.
- **34 `test` / 36 `test_most`** (optional) — `go test -cover ./...`
  gives you real numbers if you want them (~86% at time of writing).

## Fix these before you submit

Small repo blemishes that touch criteria in this form — hygiene, not
blockers:

- **Private Vulnerability Reporting is off.**  SECURITY.md's email channel
  already satisfies the vulnerability-report criteria, but the one-click
  GitHub toggle (Settings → Code security → Private vulnerability
  reporting) makes the channel unambiguous and gives reporters a
  structured flow.  Turn it on, then mention it in SECURITY.md.
- **CodeQL has no weekly schedule.**  `codeql.yml` runs on push and PR
  only; adding a `schedule:` trigger would catch findings in dormant
  branches.  `static_analysis_often` is still honestly Met on
  every-change evidence alone, so this is optional polish.
- **No `-race` in CI.**  `go test ./...` runs plain.  The fuzz targets
  already cover dynamic analysis, so nothing on the form depends on it,
  but a one-line `go test -race ./...` would strengthen the `test`
  answer for zero cost.

One more thing to keep in mind: **markdownlint, shellcheck, cue-verify,
zizmor, and actionlint are advisory gates**, run in CI but distinct from
the merge-blocking test suite.  Cite them where relevant, but cite the
gates that actually block merges (tests, CodeQL, gitleaks,
dependency-review) first.

## After you submit

- All six section counters should read full and the project page should
  flip to "in_progress" or "passing" depending on how you answered the
  SHOULD and SUGGESTED items.  (Issue #48 set the expectation at
  "in progress", which is worth 5/10 on Scorecard's CII-Best-Practices
  check; passing is a bonus.)
- Embed the badge using the snippet the project page gives you, e.g.
  `[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/XXXX/badge)](https://www.bestpractices.dev/projects/XXXX)`
  with `XXXX` replaced by the project number from the page URL.  The
  README's badge row (next to the Scorecard badge) is the natural home.
- The badge is a snapshot, not a one-time event.  Revisit the form when
  circumstances change — most importantly, if a CVE is ever assigned
  (criterion 22 flips from N/A), if vulnerability reports start arriving
  (criterion 30 flips from N/A), or if maintenance goes quiet.
