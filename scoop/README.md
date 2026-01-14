# Scoop Installation for Windows

## Quick Install

Install directly from the manifest URL:

```powershell
scoop install https://raw.githubusercontent.com/ohare93/juggle/main/scoop/juggle.json
```

## Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/ohare93/juggle/releases)
2. Extract `juggle_X.Y.Z_windows_amd64.zip`
3. Add the extracted directory to your PATH

## Updating the Manifest

After each release, update `juggle.json` with:

1. New version number
2. Updated URLs with new version
3. SHA256 hashes from `checksums.txt` in the release

Get hashes:
```powershell
# After downloading the release checksums.txt
Select-String "windows" checksums.txt
```
