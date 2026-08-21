# Releases

## For users

Download the latest version from this repository's GitHub Releases page.

Choose the archive matching your operating system and CPU:

| Platform | Asset suffix |
|---|---|
| Linux x86_64 | `linux_amd64.tar.gz` |
| Linux ARM64 | `linux_arm64.tar.gz` |
| macOS Intel | `macos_amd64.tar.gz` |
| macOS Apple Silicon | `macos_arm64.tar.gz` |
| Windows x86_64 | `windows_amd64.zip` |
| Windows ARM64 | `windows_arm64.zip` |

Every archive contains the executable, `LICENSE`, `NOTICE`, and `README.md`.
The `SHA256SUMS` asset can be used to verify the downloaded archive.

The release archive is the recommended installation path. A Go toolchain and a
local build are only needed when developing the CLI itself.

## For maintainers

After merging the changes intended for a release, create and push an annotated
semantic version tag:

```bash
git tag -a v0.0.1 -m "Release v0.0.1"
git push origin v0.0.1
```

The tag push starts `.github/workflows/release.yml`. The workflow runs tests and
lint, cross-compiles the six platform/architecture combinations, creates the
archives and `SHA256SUMS`, and creates or updates the GitHub Release with
automatically generated release notes.

Do not move an existing release tag. Publish a new patch version for a rebuilt
or corrected release.
