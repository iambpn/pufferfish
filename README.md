# Pufferfish

A cross platform clipboard manager similar to diodon.

Pufferfish lives in the system tray and records what you copy. Open the
history to put an earlier item back on the clipboard; picking an item also
moves it to the front, as if it had just been copied.

## Installation

Download the latest release from the [releases page](https://github.com/iambpn/pufferfish/releases).

> Releases are also available for Linux and Windows only. For macOS, you need to build from source.

### Linux

```sh
curl -fsSL https://raw.githubusercontent.com/iambpn/pufferfish/main/install.sh | sudo sh
```

This downloads the matching-arch release tarball and installs the binary,
`.desktop` entry, and icon under `/usr/local`. Variations:

```sh
# a specific version
curl -fsSL https://raw.githubusercontent.com/iambpn/pufferfish/main/install.sh | sudo sh -s -- v0.1.0

# uninstall
curl -fsSL https://raw.githubusercontent.com/iambpn/pufferfish/main/install.sh | sudo sh -s -- uninstall
```

Run it from a clone instead if you prefer (`sudo ./install.sh`, same
arguments). A local `*pufferfish*.tar.xz` next to the script is used in
place of a download when present. Set `PREFIX` to install somewhere other
than `/usr/local`.

## Opening the history

The tray menu has three items: **Open History**, **Clear History**, and
**Preferences**.

Running `pufferfish --history` opens the history too. If an instance is
already running it just focuses that one's window (the instances talk over
a loopback port), so the flag is safe to bind to a global hotkey. Only one
instance runs at a time; a second launch without `--history` exits.

> **NOTE:** Pufferfish has no built-in hotkey. Set up the clipboard
> manager shortcut through your system/OS keyboard settings and point it
> at the `pufferfish --history` command.

The history window has no title bar. Drag any empty part of it to move it,
and it reopens where you left it. `Esc` or the ✕ closes it.

## Preferences

| Setting | Effect |
| --- | --- |
| Use clipboard (Ctrl+C) | Start or stop tracking the system clipboard. |
| Add images to clipboard history | Also capture copied images, stored as PNG files alongside the history. |
| Keep clipboard content | Put the newest history item back on the clipboard when Pufferfish starts, so a restart doesn't lose your last copy. |
| Automatically paste selected item | Send the paste shortcut to the focused window after picking an item from the history. |
| Number of recent items | How many items the history keeps, oldest dropped first. |

Changes apply immediately and persist across restarts.

## History storage

The history index and its image files are kept in the app's storage
directory (`~/.config/fyne/pufferfish` on Linux) and reloaded on start.
Clearing the history deletes the stored images with it.

## Automatic paste

Pufferfish cannot inject a keystroke into a window it does not own, so
"automatically paste selected item" shells out to a helper.

| Platform | Helper | Needs |
| --- | --- | --- |
| Linux/BSD | `ydotool` | its `ydotoold` daemon, plus `/dev/uinput` access |
| Linux/BSD | `wtype` | a wlroots compositor — Sway, Hyprland; not GNOME or KDE |
| Linux/BSD | `xdotool` | X11; it cannot reach native Wayland windows |
| macOS | `osascript` | the Accessibility permission |
| Windows | `powershell` | nothing |

On Linux the first one found on `PATH` is used, in the order above.
`ydotool` comes first because it injects at the kernel level, which is what
makes it the only one that works on GNOME and KDE. Without any of them,
picking an item still copies it and you paste yourself.

### Linux

`ydotool` should be already installed in many popular distro and if not installed then you can install `ydotool` and enable its daemon:

```sh
sudo apt install ydotool          # or your distro's package
systemctl --user enable --now ydotoold
```

If a paste fails because the daemon is down, Pufferfish starts it — first
via `systemctl --user start ydotoold`, then by running `ydotoold` directly
— waits for it to serve, and sends the keystroke again. It never escalates
privileges, and never stops the daemon: other tools share it.

`ydotoold` needs write access to `/dev/uinput`. If it will not start, add
the udev rule below, then log out and back in:

```sh
sudo tee /etc/udev/rules.d/99-uinput.rules <<'RULE'
KERNEL=="uinput", GROUP="input", MODE="0660"
RULE
sudo usermod -aG input "$USER"
sudo udevadm control --reload-rules && sudo udevadm trigger
```

### macOS

macOS blocks synthetic keystrokes until the app is trusted. Add Pufferfish
under System Settings > Privacy & Security > Accessibility. There is no way
to grant this from inside the app.

### Windows

Works out of the box. The one limit is a Windows security rule: a program
can only send keys to windows at or below its own privilege level, so
pasting into an app running as administrator needs Pufferfish to run as
administrator too.

## Development

Run `make help` for the available targets. `make run` starts the app;
`make dev` runs it with hot reload.

The binary takes two flags: `--history` (documented above) and `--dev`,
which opens the history window on startup instead of starting hidden in
the tray.

## Releasing

The `Release` workflow (`.github/workflows/release.yml`) cross-packages
Linux (amd64, arm64) and Windows (amd64) with `fyne-cross` and publishes a
GitHub release with the archives attached and auto-generated notes.

1. Bump `Version` (and `Build`) in `FyneApp.toml`, commit, and push to
   `main`.
2. Publish, either way:
   - **Tag:** push a tag named `vX.Y.Z` (`git tag v0.1.0 && git push
     origin v0.1.0`). The release is cut for that tag.
   - **Manual:** Actions tab > Release > *Run workflow*. Leave the version
     input blank to use `v<FyneApp.toml Version>`, or type an explicit
     `vX.Y.Z`. The tag is created on the selected branch if missing.

macOS is not built in CI (Apple's SDK licence). Build it on a Mac with
`make package-darwin` and attach the result to the release manually.
