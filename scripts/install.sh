#!/usr/bin/env bash
set -euo pipefail

GITHUB_REPO="rekal-dev/cli"
DEFAULT_INSTALL_DIR="${REKAL_INSTALL_DIR:-$HOME/.local/bin}"

# Colors (disabled in non-interactive mode)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' BOLD='' NC=''
fi

info()    { printf '%b%s%b\n' "${BLUE}==>${NC} ${BOLD}" "$1" "${NC}"; }
success() { printf '%b%s%b\n' "${GREEN}==>${NC} ${BOLD}" "$1" "${NC}"; }
warn()    { printf '%b %s\n' "${YELLOW}Warning:${NC}" "$1"; }
error()   { printf '%b %s\n' "${RED}Error:${NC}" "$1" >&2; exit 1; }

detect_os() {
    case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
        darwin) echo "darwin" ;;
        linux)  echo "linux" ;;
        *)      error "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_version() {
    if [[ -n "${REKAL_VERSION:-}" ]]; then
        echo "${REKAL_VERSION#v}"
        return
    fi
    if [[ -n "${1:-}" ]]; then
        echo "${1#v}"
        return
    fi
    local url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local curl_opts=(-fsSL)
    [[ -n "${GITHUB_TOKEN:-}" ]] && curl_opts+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
    local version
    version=$(curl "${curl_opts[@]}" "$url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')
    [[ -z "$version" ]] && error "Could not fetch latest version from GitHub."
    echo "$version"
}

download_file() {
    curl -fsSL "$1" -o "$2"
}

verify_checksum() {
    local file="$1" expected="$2" actual
    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No sha256sum/shasum found. Skipping verification."
        return 0
    fi
    [[ "$actual" == "$expected" ]] || error "Checksum mismatch. Expected: $expected, got: $actual"
}

main() {
    command -v curl &>/dev/null || error "curl is required. Install curl and try again."

    # Require Claude Code — check binary on PATH or config directory.
    if ! command -v claude &>/dev/null && [[ ! -d "${HOME}/.claude" ]]; then
        echo ""
        echo -e "${RED}Error:${NC} Rekal requires Claude Code, which was not detected on this system."
        echo "For the beta release, only Claude Code is supported. Other coding agents will be supported in a future release."
        echo ""
        echo "  Install Claude Code: https://docs.anthropic.com/en/docs/claude-code"
        echo "  Rekal docs:          https://github.com/rekal-dev/cli"
        exit 1
    fi

    # Parse arguments.
    local target_dir="" version_arg=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --target)
                [[ -z "${2:-}" ]] && error "--target requires a directory path"
                target_dir="$2"; shift 2 ;;
            --target=*)
                target_dir="${1#--target=}"; shift ;;
            *)
                version_arg="$1"; shift ;;
        esac
    done

    local install_dir="${target_dir:-${DEFAULT_INSTALL_DIR}}"

    info "Installing Rekal CLI..."
    local os arch version
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    info "Resolving version..."
    version=$(get_version "${version_arg:-}")
    info "Version: ${version}"

    local archive_name="rekal_${os}_${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    REKAL_TMP_DIR="$tmp_dir"
    trap 'rm -rf "${REKAL_TMP_DIR:-}"' EXIT

    info "Downloading ${archive_name}..."
    download_file "$download_url" "${tmp_dir}/${archive_name}" || error "Download failed: $download_url"

    info "Downloading checksums..."
    download_file "$checksums_url" "${tmp_dir}/checksums.txt" || error "Failed to download checksums."

    info "Verifying checksum..."
    local expected
    expected=$(grep -E "${archive_name}\$" "${tmp_dir}/checksums.txt" | awk '{print $1}' || true)
    [[ -z "$expected" ]] && error "Checksum for ${archive_name} not found in checksums.txt."
    verify_checksum "${tmp_dir}/${archive_name}" "$expected"
    success "Checksum OK"

    info "Extracting..."
    tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

    local binary_path="${tmp_dir}/rekal"
    local install_path="${install_dir}/rekal"

    chmod +x "$binary_path"
    mkdir -p "$install_dir"
    [[ -w "$install_dir" ]] || error "Cannot write to ${install_dir}."
    mv "$binary_path" "$install_path"

    if "$install_path" version &>/dev/null; then
        success "Rekal CLI installed to ${install_path}"
    else
        error "Binary failed to run after install."
    fi

    local path_binary
    path_binary=$(command -v rekal 2>/dev/null || true)
    if [[ -n "$path_binary" && "$path_binary" != "$install_path" ]]; then
        echo ""
        echo -e "${YELLOW}!${NC} ${BOLD}PATH conflict:${NC} 'rekal' resolves to ${path_binary}, not ${install_path}"
        echo -e "  Adjust PATH or remove the other binary."
        exit 1
    fi

    if [[ -z "$path_binary" ]]; then
        echo ""
        echo -e "  ${YELLOW}Add to PATH to run \`rekal\` from anywhere:${NC}"
        echo -e "    ${BOLD}export PATH=\"${install_dir}:\$PATH\"${NC}"
        echo ""
        echo "  Then run: rekal version"
    fi
}

main "$@"
