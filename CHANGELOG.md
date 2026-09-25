# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0] - 2026-09-25

 - First tagged release: module path moved to git.bytestone.uk, build fixed, docs published

### Changed
- Module path moved to `git.bytestone.uk/hum3/gophoto`; source now hosted on the Bytestone Forgejo with GitHub as a mirror
- Standard Taskfile tasks (`fmt`, `vet`, `test`, `check`, `docs:build`, `clean`); gokrazy tasks renamed `gok:add`, `gok:update`, `gok:image`
- Documentation published at https://gophoto.docs.bytestone.uk/

### Fixed
- `drawing.ScaleImageOuter` returned an empty rectangle: the scale ratio was read before it was set, and centring used the image size instead of the target size
- `prisimdemo` and `x11PrismSimpleDemo` did not compile against the current `frame` API
