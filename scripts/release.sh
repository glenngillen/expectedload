#!/bin/sh
# Builds the release artifacts for the expected-load parser plugin:
# per-platform archives plus a SHA-256 checksums file, all in dist/.
#
# Plain POSIX shell + go + tar/zip; runnable locally on macOS and Linux.
# Publishing (CI, GitHub Releases, plugin registry) is a later stage that
# calls this script — see specs/release-script.md.
set -eu

cd "$(dirname "$0")/.."

BINARY=infracost-parser-plugin-expectedload
DIST=dist
PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

VERSION=$(./scripts/version.sh)
case "$VERSION" in
*-dev+*)
    echo "warning: building from an untagged or dirty tree; version is $VERSION" >&2
    ;;
esac

echo "==> running tests"
go test ./...

echo "==> building $BINARY $VERSION"
rm -rf "$DIST"
mkdir -p "$DIST"

have_zip=1
command -v zip >/dev/null 2>&1 || have_zip=0

for platform in $PLATFORMS; do
    os=${platform%/*}
    arch=${platform#*/}
    ext=""
    [ "$os" = "windows" ] && ext=".exe"

    stage="$DIST/${BINARY}_${VERSION}_${os}_${arch}"
    echo "==> $os/$arch"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
        -o "$stage/$BINARY$ext" ./cmd/$BINARY

    cp LICENSE README.md "$stage/"

    # The binary keeps its exact SDK-mandated name inside the archive; the
    # archive name alone carries the version/platform disambiguation.
    if [ "$os" = "windows" ] && [ "$have_zip" = "1" ]; then
        (cd "$stage" && zip -q "../$(basename "$stage").zip" "$BINARY$ext" LICENSE README.md)
    else
        if [ "$os" = "windows" ]; then
            echo "warning: 'zip' not found; packaging windows artifact as .tar.gz" >&2
        fi
        tar -czf "$stage.tar.gz" -C "$stage" "$BINARY$ext" LICENSE README.md
    fi
    rm -rf "$stage"
done

echo "==> writing checksums"
(
    cd "$DIST"
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || shasum -a 256 ./*.tar.gz > checksums.txt
    else
        sha256sum ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || sha256sum ./*.tar.gz > checksums.txt
    fi
)

echo "==> done"
ls -l "$DIST"
