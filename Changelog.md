# Change Log
<!---
Always update Version in Makefile
-->

## Unreleased

## [v1.4.4] - 2022-10-24

### Changed
- updated Building.md
- updated Readme.md
- Scripts now get the full filepath as second argument (before it would pass `bash` now it will pass `/bin/bash`)

### Fixed
- Fix a bug where incorrect keys in checksec were silently skipped

## [v1.4.3] - 2020-08-17

### Changed
- support older versions of checksec

## [v1.4.2] - 2020-08-17

### Added
- checksec wrapper script, see [check_sec.sh](scripts/check_sec.sh) and [Checksec Wrapper Readme](Checksec.md)
- link support for extfs, this requires `https://github.com/crmulliner/e2tools/tree/link_support` (or later)

### Changed
- updated `test/test.img.gz` ext2 test filesystem image
- updated `test/e2cp` binary

## [v1.4.1] - 2020-05-06

### Fixed
- removed `release/` folder
- FileStatCheck for links
- general handling for links

## [v1.4.0] - 2020-04-30

### Added
- NEW support for Linux Capabilities
- NEW Capability support for ext2/3/4 and squashfs
- NEW Selinux support for SquashFS

### Changed
- _check.py_ cleaned up a bit, avoiding using `shell=True` in subprocess invocations.
- updated linter version to v1.24
- switch back to `-lls` for unsquashfs
- copyright: GM Cruise -> Cruise

### Fixed
- FileTreeCheck LinkTarget handling

## [v1.3.2] - 2020-01-15

### Fixed
- _check.py_ fix to support pathnames with spaces
- _cpiofs_ fix date parsing
- _cpiofs_ added work around for missing directory entries

## [v1.3.1] - 2020-01-07

### Fixed
- report status in _check.py_
- use quiet flag for _cpiofs_

## [v1.3.0] - 2020-01-07

### Added
- NEW _cpiofs_ for cpio as filesystem
- NEW universal _check.py_ (so you just need to write a custom unpacker)
- NEW _android/unpack.sh_ (for _check.py_)
- better options for scripts (FileContent and DataExtract)

### Fixed
- $PATH in makefile
- FileContent file iterator
- _squashfs_ username parsing

## [v1.2.0] - 2019-11-19

### Changed
- moved to go 1.13
- only store _current_file_treepath_ if filetree changed

## [v.1.1.0] - 2019-10-15

### Added
- NEW FileCmp check for full file diff against 'old' version
- allow multiple matches for regex based DataExtract

### Fixed
- squashfs username parsing

## [v.1.0.1] - 2019-09-19

### Fixed
- filename for BadFiles check output

## [v.1.0.0] - 2019-08-15

### Added
- CI
- Build instructions

## [initial] - 2019-08-05
