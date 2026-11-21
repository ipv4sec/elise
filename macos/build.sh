#!/bin/bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
output="${1:-$root/dist/Elise.app}"
if [[ "$output" != *.app || -e "$output" || -L "$output" ]]; then
    printf 'Choose a new output path ending in .app: %s\n' "$output" >&2
    exit 1
fi

parent="$(dirname "$output")"
mkdir -p "$parent"
parent="$(cd "$parent" && pwd)"
stage="$(mktemp -d "$parent/.elise-build.XXXXXX")"
trap 'rm -rf -- "$stage"' EXIT

bundle="$stage/$(basename "$output")"
mkdir -p "$bundle/Contents/MacOS"
cp "$root/macos/Info.plist" "$bundle/Contents/Info.plist"
/usr/bin/plutil -lint "$bundle/Contents/Info.plist"

cd "$root"
export MACOSX_DEPLOYMENT_TARGET
MACOSX_DEPLOYMENT_TARGET="$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' macos/Info.plist)"
CGO_ENABLED=1 go build -trimpath -o "$bundle/Contents/MacOS/Elise" ./cmd
/usr/bin/codesign --sign - --timestamp=none "$bundle"
/usr/bin/codesign --verify --strict "$bundle"
destination="$parent/$(basename "$output")"
/bin/mv -n "$bundle" "$parent/"
if [[ -e "$bundle" ]]; then
    printf 'The output path already exists: %s\n' "$destination" >&2
    exit 1
fi
printf '%s\n' "$destination"
