package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/iambpn/pufferfish/internal/preferences"
)

// NewClipboardSection builds the clipboard preferences form.
func NewClipboardSection(prefs *preferences.ClipboardPreferences) fyne.CanvasObject {
	useClipboard := widget.NewCheck("Use clipboard (Ctrl+C)", prefs.SetUseClipboard)
	useClipboard.SetChecked(prefs.UseClipboard)

	addImages := widget.NewCheck("Add images to clipboard history", prefs.SetAddImages)
	addImages.SetChecked(prefs.AddImages)

	keepContent := widget.NewCheck("Keep clipboard content", prefs.SetKeepContent)
	keepContent.SetChecked(prefs.KeepContent)

	autoPaste := widget.NewCheck("Automatically paste selected item", prefs.SetAutoPaste)
	autoPaste.SetChecked(prefs.AutoPaste)

	recentItems := container.NewHBox(
		container.NewCenter(widget.NewLabel("Number of recent items")),
		layout.NewSpacer(),
		newNumberStepper(prefs.RecentItems, preferences.MinRecentItems, preferences.MaxRecentItems, prefs.SetRecentItems),
	)

	historyPosition := container.NewHBox(
		container.NewCenter(widget.NewLabel("History window position")),
		layout.NewSpacer(),
		newHistoryPositionSelect(prefs),
	)

	tip := newFloatingTip()
	rows := container.NewVBox(
		withTooltip(tip, useClipboard, "Watch the clipboard for new copies made with Ctrl+C."),
		withTooltip(tip, addImages, "Also save copied images to the clipboard history."),
		withTooltip(tip, keepContent, "Keep the last copied item in the system clipboard after Pufferfish closes."),
		withTooltip(tip, autoPaste, "Automatically paste the selected history item into the active app."),
		withTooltip(tip, recentItems, "The maximum number of recent items kept in the clipboard history."),
		withTooltip(tip, historyPosition, "Where the clipboard history window opens on screen."),
	)

	return container.NewPadded(container.NewStack(rows, tip))
}

// newNumberStepper renders a numeric value with -/+ buttons, calling
// onChange whenever the value settles on a new number.
func newNumberStepper(value, min, max int, onChange func(int)) fyne.CanvasObject {
	current := value

	valueLabel := widget.NewLabel(strconv.Itoa(current))
	valueLabel.Alignment = fyne.TextAlignCenter

	setValue := func(v int) {
		if v < min {
			v = min
		}
		if v > max {
			v = max
		}
		current = v
		valueLabel.SetText(strconv.Itoa(current))
		onChange(current)
	}

	minusBtn := newCircleIconButton(theme.ContentRemoveIcon(), func() { setValue(current - 1) })
	plusBtn := newCircleIconButton(theme.ContentAddIcon(), func() { setValue(current + 1) })

	return container.NewHBox(
		minusBtn,
		container.NewCenter(valueLabel),
		plusBtn,
	)
}

// newHistoryPositionSelect renders a dropdown of screen anchors for the
// history window, calling prefs.SetHistoryPosition when the user picks one.
func newHistoryPositionSelect(prefs *preferences.ClipboardPreferences) fyne.CanvasObject {
	labels := make([]string, len(preferences.HistoryAnchors))
	valueForLabel := map[string]preferences.HistoryPosition{}
	initialLabel := ""
	for i, a := range preferences.HistoryAnchors {
		labels[i] = a.Label
		valueForLabel[a.Label] = a.Position
		if a.Position == prefs.HistoryPosition {
			initialLabel = a.Label
		}
	}

	sel := widget.NewSelect(labels, func(label string) {
		prefs.SetHistoryPosition(valueForLabel[label])
	})
	sel.SetSelected(initialLabel)

	return sel
}
