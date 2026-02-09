#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
DRY_RUN=""

if [ -z "$VERSION" ]; then
  echo "Usage: ./publish.sh <version> [--dry-run]"
  echo "Example: ./publish.sh 0.3.0 --dry-run"
  exit 1
fi

if [ "${2:-}" = "--dry-run" ]; then
  DRY_RUN="--dry-run"
  echo "==> Dry run mode enabled"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_DIR="$(dirname "$SCRIPT_DIR")"
NPM_DIR="$SCRIPT_DIR"

# Go build settings
MODULE="github.com/tradeboba/boba-cli/internal/version"
COMMIT="$(git -C "$GO_DIR" rev-parse --short HEAD 2>/dev/null || echo "none")"
DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
LDFLAGS="-s -w -X ${MODULE}.Version=${VERSION} -X ${MODULE}.Commit=${COMMIT} -X ${MODULE}.Date=${DATE}"

# Platform targets: "GOOS GOARCH npm-package-suffix binary-name"
TARGETS=(
  "darwin arm64 cli-darwin-arm64 boba"
  "darwin amd64 cli-darwin-x64 boba"
  "linux  amd64 cli-linux-x64 boba"
  "linux  arm64 cli-linux-arm64 boba"
  "windows amd64 cli-win32-x64 boba.exe"
)

echo "==> Cross-compiling Go binaries for v${VERSION}..."

for target in "${TARGETS[@]}"; do
  read -r goos goarch pkg_suffix binary_name <<< "$target"
  out_dir="${NPM_DIR}/@tradeboba/${pkg_suffix}/bin"
  mkdir -p "$out_dir"

  echo "    ${goos}/${goarch} -> ${pkg_suffix}/bin/${binary_name}"
  GOOS="$goos" GOARCH="$goarch" go build \
    -C "$GO_DIR" \
    -ldflags "$LDFLAGS" \
    -o "${out_dir}/${binary_name}" \
    ./cmd/boba
done

echo "==> Updating versions to ${VERSION}..."

# Update all platform package.json files
for target in "${TARGETS[@]}"; do
  read -r _ _ pkg_suffix _ <<< "$target"
  pkg_json="${NPM_DIR}/@tradeboba/${pkg_suffix}/package.json"
  # Use node for portable JSON editing
  node -e "
    const fs = require('fs');
    const pkg = JSON.parse(fs.readFileSync('${pkg_json}', 'utf8'));
    pkg.version = '${VERSION}';
    fs.writeFileSync('${pkg_json}', JSON.stringify(pkg, null, 2) + '\n');
  "
done

# Update main wrapper package.json (version + optionalDependencies versions)
WRAPPER_JSON="${NPM_DIR}/@tradeboba/cli/package.json"
node -e "
  const fs = require('fs');
  const pkg = JSON.parse(fs.readFileSync('${WRAPPER_JSON}', 'utf8'));
  pkg.version = '${VERSION}';
  for (const dep of Object.keys(pkg.optionalDependencies || {})) {
    pkg.optionalDependencies[dep] = '${VERSION}';
  }
  fs.writeFileSync('${WRAPPER_JSON}', JSON.stringify(pkg, null, 2) + '\n');
"

echo "==> Publishing platform packages..."

for target in "${TARGETS[@]}"; do
  read -r _ _ pkg_suffix _ <<< "$target"
  pkg_dir="${NPM_DIR}/@tradeboba/${pkg_suffix}"
  echo "    npm publish ${pkg_suffix} ${DRY_RUN}"
  npm publish "$pkg_dir" --access public $DRY_RUN
done

echo "==> Publishing main wrapper package..."
npm publish "${NPM_DIR}/@tradeboba/cli" --access public $DRY_RUN

echo "==> Cleaning up binaries..."

for target in "${TARGETS[@]}"; do
  read -r _ _ pkg_suffix _ <<< "$target"
  rm -rf "${NPM_DIR}/@tradeboba/${pkg_suffix}/bin"
done

echo "==> Done! Published @tradeboba/cli v${VERSION}"
