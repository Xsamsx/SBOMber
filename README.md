# SBOMber

SBOMber is an open-source Go CLI for scanning directories of locally cloned Git repositories and generating software bill of materials (SBOM) artifacts at scale.

The project starts with multi-repository discovery and SBOM generation, then expands into dependency metadata, outdated package detection, and supply-chain analysis.

## Why SBOMber

- Scan a parent folder that contains many repositories
- Detect repositories recursively instead of handling one repo at a time
- Build toward support for `npm`, `Python`, `Maven`, `Ruby`, and `Go`
- Generate standards-based SBOM output such as `CycloneDX` and `SPDX`
- Keep the tool scriptable, local-first, and friendly to automation

## Current Status

The repository is scaffolded as a production-ready Go project with:

- a working CLI entrypoint
- recursive Git repository discovery
- CI for formatting, vetting, and tests
- OSS community files for contributions and issue reporting

SBOM extraction backends are the next implementation step.

## Quick Start

### Prerequisites

- Go `1.26` or newer

### Run locally

```bash
make tidy
make build
./bin/sbomber scan ../
```

### Run without building

```bash
go run ./cmd/sbomber scan ../
```

## Example Output

```text
Found 1 repository under ../
- SBOMber  /absolute/path/to/SBOMber
```

## Roadmap

- Repository discovery and workspace scanning
- Ecosystem detection from manifests and lockfiles
- SBOM generation for supported stacks
- Dependency metadata and outdated package reporting
- Vulnerability and supply-chain analysis

## Project Layout

```text
cmd/sbomber/        CLI entrypoint
internal/cli/       argument parsing and command execution
internal/discovery/ repository scanning logic
.github/            CI and community health files
```

## Development

```bash
make fmt
make test
make vet
make ci
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for local development guidance and contribution workflow.

## License

Licensed under [Apache-2.0](./LICENSE).
