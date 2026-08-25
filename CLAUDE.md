# Pufferfish

## Testing constraints

- Do not attempt to visually test this app in this environment: no screenshot path works here (X11 image grab is denied, Wayland screencopy is unsupported by the compositor, and the GNOME Shell D-Bus screenshot API refuses). Don't try `import`, `grim`, `gdbus`/`org.gnome.Shell.Screenshot`, or similar - it will fail and just burn time and tokens.
- Also skip real-cursor/real-window-manager positioning experiments (moving the mouse, launching the app to check where a window physically lands, wmctrl-based repositioning tests). This machine's GNOME/Mutter session has already shown unreliable, inconsistent window-placement behavior when probed this way, so results from it don't generalize.
- Verify UI and window-placement logic with unit tests instead (Fyne's `test` package, checking computed values/colors/widget state), and rely on `go build`/`go vet`/`go test ./...` for correctness. If a change genuinely needs eyes-on visual confirmation, say so and let the user check rather than trying to capture it yourself.
