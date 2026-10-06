# Contributing

Thanks for considering a contribution to Codex Feishu Relay. Please open an issue before starting a large change so maintainers can agree on the scope and user-facing behavior.

## Development setup

- Go version: use the version declared by `scripts/ci/go-version-spec.sh`.
- Node.js: use an active LTS release for the web admin UI.
- Install web dependencies with `npm ci` from `web/`.
- Read [DEVELOPER.md](./DEVELOPER.md) and the relevant documents under [`docs/`](./docs/README.md) before changing architecture or protocol behavior.

## Change workflow

1. Keep pull requests focused and explain the user-visible problem they solve.
2. Add or update tests when behavior changes; preserve compatibility across Codex, Claude Code, Feishu, and supported operating systems where applicable.
3. Update user or developer documentation when commands, configuration, permissions, installation, or behavior changes.
4. Run the checks relevant to your change. The full CI workflow runs `go test ./...`, web tests, formatting, builds, and installer checks; `make check` is the local aggregate entry point.
5. Never commit credentials, local configuration, personal data, generated release artifacts, or machine-specific paths.

## Pull requests

Describe the motivation, behavior change, compatibility impact, and validation performed. Include screenshots for visible UI changes. Call out any configuration migration or security implications.

By submitting a contribution, you confirm that you have the right to submit it under the MIT License. Contributions are provided under the terms in [LICENSE](./LICENSE).
