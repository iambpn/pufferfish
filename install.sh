#!/bin/sh
# Install or uninstall a Pufferfish Linux release build.
#
#   curl -fsSL .../install.sh | sudo sh                    install latest
#   curl -fsSL .../install.sh | sudo sh -s -- v0.1.0       install a version
#   curl -fsSL .../install.sh | sudo sh -s -- uninstall    remove it
#
#   sudo ./install.sh [ARCHIVE|VERSION|uninstall]          same, run locally
#
# With no ARCHIVE the script downloads the matching-arch release tarball
# from GitHub. A local *pufferfish*.tar.xz next to the script is used
# instead when present. Env overrides: PREFIX (default /usr/local),
# PUFFERFISH_REPO (default iambpn/pufferfish), PUFFERFISH_VERSION,
# AUTOSTART_DIR (default /etc/xdg/autostart).
#
# Installing also drops a system-wide XDG autostart entry so the tray app
# starts on login for every user; uninstall removes it.

set -eu

APP=pufferfish
PREFIX="${PREFIX:-/usr/local}"
REPO="${PUFFERFISH_REPO:-iambpn/pufferfish}"
AUTOSTART_DIR="${AUTOSTART_DIR:-/etc/xdg/autostart}"

BIN_DST="$PREFIX/bin/$APP"
DESKTOP_DST="$PREFIX/share/applications/$APP.desktop"
ICON_DST="$PREFIX/share/pixmaps/$APP.png"
AUTOSTART_DST="$AUTOSTART_DIR/$APP.desktop"

die() { echo "install.sh: $*" >&2; exit 1; }

usage() {
	cat <<EOF
Install or uninstall a Pufferfish Linux release build.

  curl -fsSL .../install.sh | sudo sh                    install latest
  curl -fsSL .../install.sh | sudo sh -s -- v0.1.0       install a version
  curl -fsSL .../install.sh | sudo sh -s -- uninstall    remove it

  sudo ./install.sh [ARCHIVE|VERSION|uninstall]          same, run locally

Env: PREFIX (default /usr/local), PUFFERFISH_REPO, PUFFERFISH_VERSION,
     AUTOSTART_DIR (default /etc/xdg/autostart).
EOF
}

need_writable() {
	for sub in bin share/applications share/pixmaps; do
		mkdir -p "$PREFIX/$sub" 2>/dev/null || true
	done
	{ [ -w "$PREFIX/bin" ] && [ -w "$PREFIX/share/applications" ] && [ -w "$PREFIX/share/pixmaps" ]; } ||
		die "cannot write under $PREFIX; re-run with sudo (or set PREFIX to a writable dir)"
}

refresh_desktop_db() {
	command -v update-desktop-database >/dev/null 2>&1 || return 0
	update-desktop-database "$PREFIX/share/applications" 2>/dev/null || true
}

install_autostart() {
	mkdir -p "$AUTOSTART_DIR" 2>/dev/null || true
	[ -w "$AUTOSTART_DIR" ] || {
		echo "install.sh: $AUTOSTART_DIR not writable; skipping autostart entry" >&2
		return 0
	}
	cat >"$AUTOSTART_DST" <<EOF
[Desktop Entry]
Type=Application
Name=Pufferfish
Comment=Clipboard manager
Exec=$BIN_DST
Icon=$ICON_DST
Terminal=false
Categories=Utility;
X-GNOME-Autostart-enabled=true
EOF
	chmod 644 "$AUTOSTART_DST"
	echo "autostart entry -> $AUTOSTART_DST"
}

remove_autostart() {
	rm -f "$AUTOSTART_DST"
}

resolve_arch() {
	case "$(uname -m)" in
		x86_64|amd64) echo amd64 ;;
		aarch64|arm64) echo arm64 ;;
		*) die "unsupported arch: $(uname -m)" ;;
	esac
}

fetch_stdout() {
	if command -v curl >/dev/null 2>&1; then curl -fsSL "$1"
	elif command -v wget >/dev/null 2>&1; then wget -qO- "$1"
	else die "need curl or wget to download"; fi
}

fetch_file() {
	if command -v curl >/dev/null 2>&1; then curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then wget -qO "$2" "$1"
	else die "need curl or wget to download"; fi
}

latest_tag() {
	fetch_stdout "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1
}

find_local_archive() {
	want="$(resolve_arch)"
	dir="$(dirname "$0")"
	for f in "$dir"/*"$want"*pufferfish*.tar.xz "$dir"/*pufferfish*.tar.xz; do
		[ -f "$f" ] && { echo "$f"; return 0; }
	done
	return 1
}

download_archive() {
	dest="$1"
	arch="$(resolve_arch)"
	tag="${PUFFERFISH_VERSION:-}"
	[ -n "$tag" ] || tag="$(latest_tag)"
	[ -n "$tag" ] || die "could not resolve the latest release tag"
	name="linux-$arch-$APP.tar.xz"
	url="https://github.com/$REPO/releases/download/$tag/$name"
	echo "downloading $url" >&2
	fetch_file "$url" "$dest/$name" || die "download failed: $url"
	echo "$dest/$name"
}

do_install() {
	need_writable
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT

	archive="${1:-}"
	if [ -n "$archive" ] && [ ! -f "$archive" ]; then
		case "$archive" in
			v[0-9]*) PUFFERFISH_VERSION="$archive"; archive= ;;
			*) die "archive not found: $archive" ;;
		esac
	fi
	[ -n "$archive" ] || archive="$(find_local_archive || true)"
	[ -n "$archive" ] || archive="$(download_archive "$tmp")"

	ex="$tmp/x"; mkdir -p "$ex"
	tar -xJf "$archive" -C "$ex"

	bin_src="$(find "$ex" -type f -path "*/bin/$APP" -print -quit)"
	[ -n "$bin_src" ] || die "no $APP binary inside $(basename "$archive")"
	desktop_src="$(find "$ex" -type f -name "$APP.desktop" -print -quit)"
	icon_src="$(find "$ex" -type f -name "$APP.png" -print -quit)"

	install -Dm755 "$bin_src" "$BIN_DST"
	[ -n "$desktop_src" ] && install -Dm644 "$desktop_src" "$DESKTOP_DST" || true
	[ -n "$icon_src" ] && install -Dm644 "$icon_src" "$ICON_DST" || true
	refresh_desktop_db
	install_autostart

	echo "installed $APP -> $BIN_DST (from $(basename "$archive"))"
}

do_uninstall() {
	need_writable
	rm -f "$BIN_DST" "$DESKTOP_DST" "$ICON_DST"
	remove_autostart
	refresh_desktop_db
	echo "removed $APP from $PREFIX"
}

case "${1:-install}" in
	uninstall|remove) do_uninstall ;;
	install)          do_install "${2:-}" ;;
	-h|--help)        usage ;;
	*)                do_install "$1" ;;
esac
