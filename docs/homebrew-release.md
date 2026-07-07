# Homebrew Release Guide

This guide sets Wagon up for a Homebrew tap release under OverStackedLab.

## Repositories

Use two GitHub repositories:

- `OverStackedLab/wagon`: source code and release tags.
- `OverStackedLab/homebrew-tap`: Homebrew tap repository.

Homebrew refers to `OverStackedLab/homebrew-tap` as `OverStackedLab/tap`, because tap repositories use the `homebrew-` prefix on GitHub but omit it in `brew` commands.

## Current Release Setup

The source repo is configured for the OverStackedLab package identity:

```text
module github.com/OverStackedLab/wagon
```

Release builds can inject the tagged version into `wagon version`:

```bash
make release-check VERSION=0.1.0
```

The Homebrew formula template lives at:

```text
packaging/homebrew/wagon.rb
```

It intentionally contains a placeholder checksum because GitHub creates the release tarball after the tag exists.

Copy that file into the tap repo as:

```text
Formula/wagon.rb
```

## One-Time Tap Setup

Create the tap repository on GitHub:

```text
OverStackedLab/homebrew-tap
```

Clone it locally:

```bash
git clone git@github.com:OverStackedLab/homebrew-tap.git ../homebrew-tap
mkdir -p ../homebrew-tap/Formula
```

## Release Steps

From the Wagon repo:

```bash
go test ./...
make release-check VERSION=0.1.0
git status --short
git tag v0.1.0
git push origin main
git push origin v0.1.0
```

Create a GitHub release for `v0.1.0` in `OverStackedLab/wagon`.

Compute the release tarball checksum:

```bash
curl -L -o /tmp/wagon-v0.1.0.tar.gz \
  https://github.com/OverStackedLab/wagon/archive/refs/tags/v0.1.0.tar.gz
shasum -a 256 /tmp/wagon-v0.1.0.tar.gz
```

Copy the formula template into the tap:

```bash
cp packaging/homebrew/wagon.rb ../homebrew-tap/Formula/wagon.rb
```

Edit `../homebrew-tap/Formula/wagon.rb` and replace:

```text
REPLACE_WITH_RELEASE_TARBALL_SHA256
```

with the checksum from `shasum`.

From the tap repo:

```bash
brew install --build-from-source --verbose ./Formula/wagon.rb
brew test ./Formula/wagon.rb
brew audit --strict --new --online ./Formula/wagon.rb
git add Formula/wagon.rb
git commit -m "Add wagon formula"
git push origin main
```

## User Install Commands

Direct install:

```bash
brew install OverStackedLab/tap/wagon
```

Manual tap install:

```bash
brew tap OverStackedLab/tap
brew install wagon
```

## Updating Later Releases

For `v0.2.0` and later:

1. Tag and push the new source release in `OverStackedLab/wagon`.
2. Download the new tag tarball and compute its SHA-256 checksum.
3. Update `url` and `sha256` in `OverStackedLab/homebrew-tap/Formula/wagon.rb`.
4. Run `brew install --build-from-source --verbose ./Formula/wagon.rb`.
5. Run `brew test ./Formula/wagon.rb`.
6. Run `brew audit --strict --online ./Formula/wagon.rb`.
7. Commit and push the tap change.

## References

- Homebrew tap guide: https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap
- Homebrew formula cookbook: https://docs.brew.sh/Formula-Cookbook
- Homebrew Formula API: https://docs.brew.sh/rubydoc/Formula.html
