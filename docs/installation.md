# Installation

dygo release binaries are distributed through GitHub Releases.

Install the latest release:

```sh
curl -fsSL https://dygo.dev/install | sh
```

The installer places the managed binary in:

```txt
~/.dygo/bin
```

If that directory is not on `PATH`, the installer prints the shell profile line to add.

## Options

Install a specific version:

```sh
curl -fsSL https://dygo.dev/install | DYGO_VERSION=v0.1.0 sh
```

Install somewhere else:

```sh
curl -fsSL https://dygo.dev/install | DYGO_INSTALL_DIR=/usr/local/bin sh
```

Windows PowerShell:

```powershell
irm https://dygo.dev/install.ps1 | iex
```

## Upgrades

Update the dygo binary out of band with the installer:

```sh
curl -fsSL https://dygo.dev/install | sh
```

Inside a generated dygo project, `dygo upgrade` updates the project `go.mod` dygo dependency, dygo-managed generated runner files, and the cached Studio UI assets when the target dygo version differs from the project version. Project upgrades refuse dirty git worktrees.

Useful upgrade modes:

```sh
dygo upgrade --check
dygo upgrade --dry-run
dygo upgrade --to v0.1.0
dygo upgrade --yes
```

The installers verify the archive checksum and the embedded dygo version before they replace the managed binary.

Maintainers should use the [Release Process](releasing.md) to build, tag, and publish framework releases.
