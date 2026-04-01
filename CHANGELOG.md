# Changelog

## 1.1.0

### Changed

- Renamed the config flag from `-config` to `-c`.
- Updated CLI help formatting to show `netmap [-h]`, add a blank line after `Usage:`, and simplify the `Output:` section.
- `netmap` with no arguments and `netmap -h` continue to print help without attempting ADC credential or provider initialization.
- GitHub releases now use `vVERSION` as both the release tag and release title.
- The embedded CLI version, repository `VERSION`, and release documentation now point to `1.1.0`.
