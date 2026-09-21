#!/bin/sh

set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

package="$tmp/package"
prefix="$tmp/prefix"
autostart_dir="$tmp/autostart"
fake_bin="$tmp/fake-bin"
mkdir -p "$package/usr/local/share/applications" \
	"$package/usr/local/share/pixmaps" "$fake_bin" "$autostart_dir"

cat >"$package/pufferfish" <<'EOF'
#!/bin/sh
EOF
chmod 755 "$package/pufferfish"

cat >"$package/usr/local/share/applications/com.iambpn.pufferfish.desktop" <<'EOF'
[Desktop Entry]
Name=pufferfish
Exec=pufferfish
Icon=pufferfish
Type=Application
NoDisplay=false
X-GNOME-Autostart-enabled=false

[X-Fyne Source]
Repo=https://github.com/iambpn/pufferfish
EOF

: >"$package/usr/local/share/pixmaps/pufferfish.png"

cat >"$package/Makefile" <<'EOF'
install:
	mkdir -p "$(PREFIX)/bin" "$(PREFIX)/share/applications" "$(PREFIX)/share/pixmaps"
	cp pufferfish "$(PREFIX)/bin/pufferfish"
	cp usr/local/share/applications/com.iambpn.pufferfish.desktop "$(PREFIX)/share/applications/"
	cp usr/local/share/pixmaps/pufferfish.png "$(PREFIX)/share/pixmaps/"
EOF

# Keep the installer from touching real users' Desktop directories.
cat >"$fake_bin/getent" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 755 "$fake_bin/getent"

archive="$tmp/linux-amd64-pufferfish.tar.xz"
tar -cJf "$archive" -C "$package" .

# Simulate upgrading an install that used the old desktop ID.
: >"$autostart_dir/pufferfish.desktop"
PATH="$fake_bin:$PATH" PREFIX="$prefix" AUTOSTART_DIR="$autostart_dir" \
	sh "$repo_dir/install.sh" "$archive"

entry="$autostart_dir/com.iambpn.pufferfish.desktop"
test -f "$entry"
test ! -e "$autostart_dir/pufferfish.desktop"
grep -q "^Exec=$prefix/bin/pufferfish$" "$entry"
test "$(grep -c '^NoDisplay=true$' "$entry")" -eq 1
test "$(grep -c '^X-GNOME-Autostart-enabled=true$' "$entry")" -eq 1
test "$(grep -c '^NoDisplay=' "$entry")" -eq 1
test "$(grep -c '^X-GNOME-Autostart-enabled=' "$entry")" -eq 1
test "$(sed -n '2p' "$entry")" = "NoDisplay=true"
test "$(sed -n '3p' "$entry")" = "X-GNOME-Autostart-enabled=true"

# Uninstall cleans up both the current and legacy names.
: >"$autostart_dir/pufferfish.desktop"
PATH="$fake_bin:$PATH" PREFIX="$prefix" AUTOSTART_DIR="$autostart_dir" \
	sh "$repo_dir/install.sh" uninstall
test ! -e "$entry"
test ! -e "$autostart_dir/pufferfish.desktop"

echo "install test passed"
