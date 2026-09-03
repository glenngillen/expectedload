#!/bin/sh
# Resolves the version to stamp into the plugin binary:
#   1. $VERSION, when set
#   2. the exact git tag, when HEAD is tagged and the tree is clean
#   3. <last-tag or v0.0.0>-dev+<short-sha> otherwise (dirty/untagged trees
#      never fail — dev builds are useful for local testing)
set -eu

if [ -n "${VERSION:-}" ]; then
    echo "$VERSION"
    exit 0
fi

if git diff-index --quiet HEAD -- 2>/dev/null; then
    if tag=$(git describe --tags --exact-match 2>/dev/null); then
        echo "$tag"
        exit 0
    fi
fi

last=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
sha=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
echo "${last}-dev+${sha}"
