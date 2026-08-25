package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/iambpn/pufferfish/internal/preferences"
)

// walk descends through containers and widget renderers, calling visit for
// every object found.
func walk(obj fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(obj)
	switch t := obj.(type) {
	case *fyne.Container:
		for _, child := range t.Objects {
			walk(child, visit)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(t).Objects() {
			walk(child, visit)
		}
	}
}

func checkByLabel(root fyne.CanvasObject, label string) *widget.Check {
	var found *widget.Check
	walk(root, func(o fyne.CanvasObject) {
		if c, ok := o.(*widget.Check); ok && c.Text == label {
			found = c
		}
	})
	return found
}

func selectWidget(root fyne.CanvasObject) *widget.Select {
	var found *widget.Select
	walk(root, func(o fyne.CanvasObject) {
		if s, ok := o.(*widget.Select); ok {
			found = s
		}
	})
	return found
}

func circleButtons(root fyne.CanvasObject) []*circleIconButton {
	var found []*circleIconButton
	walk(root, func(o fyne.CanvasObject) {
		if b, ok := o.(*circleIconButton); ok {
			found = append(found, b)
		}
	})
	return found
}

func TestClipboardSectionReflectsCurrentPreferences(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)
	prefs.SetUseClipboard(false)

	section := NewClipboardSection(prefs)

	check := checkByLabel(section, "Use clipboard (Ctrl+C)")
	if check == nil {
		t.Fatal("checkbox not found")
	}
	if check.Checked {
		t.Fatal("checkbox should reflect the disabled preference")
	}
}

func TestClipboardSectionCheckboxTogglesThePreference(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)

	section := NewClipboardSection(prefs)

	check := checkByLabel(section, "Add images to clipboard history")
	if check == nil {
		t.Fatal("checkbox not found")
	}

	check.SetChecked(false)
	if prefs.AddImages {
		t.Fatal("unchecking the box should update the preference")
	}
}

func TestHistoryPositionSelectReflectsCurrentPreference(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)
	prefs.SetHistoryPosition(preferences.HistoryPositionBottomLeft)

	section := NewClipboardSection(prefs)

	sel := selectWidget(section)
	if sel == nil {
		t.Fatal("select not found")
	}
	if sel.Selected != "Bottom Left" {
		t.Fatalf("selected = %q", sel.Selected)
	}
}

func TestHistoryPositionSelectUpdatesThePreference(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)

	section := NewClipboardSection(prefs)

	sel := selectWidget(section)
	if sel == nil {
		t.Fatal("select not found")
	}

	sel.SetSelected("Top Right")
	if prefs.HistoryPosition != preferences.HistoryPositionTopRight {
		t.Fatalf("history position = %q", prefs.HistoryPosition)
	}
}

func TestNumberStepperClampsToMinAndMax(t *testing.T) {
	test.NewTempApp(t)
	var got int
	stepper := newNumberStepper(preferences.MinRecentItems, preferences.MinRecentItems, preferences.MaxRecentItems,
		func(v int) { got = v })

	buttons := circleButtons(stepper)
	if len(buttons) != 2 {
		t.Fatalf("want 2 buttons (minus/plus), got %d", len(buttons))
	}
	minusBtn, plusBtn := buttons[0], buttons[1]

	test.Tap(minusBtn)
	if got != preferences.MinRecentItems {
		t.Fatalf("stepping below the minimum should clamp, got %d", got)
	}

	for i := 0; i < preferences.MaxRecentItems+5; i++ {
		test.Tap(plusBtn)
	}
	if got != preferences.MaxRecentItems {
		t.Fatalf("stepping above the maximum should clamp, got %d", got)
	}
}

func TestNumberStepperUpdatesTheDisplayedValue(t *testing.T) {
	test.NewTempApp(t)
	stepper := newNumberStepper(5, 1, 10, func(int) {})

	buttons := circleButtons(stepper)
	plusBtn := buttons[1]

	label := stepperLabel(stepper)
	if label.Text != "5" {
		t.Fatalf("initial label = %q", label.Text)
	}

	test.Tap(plusBtn)
	if label.Text != "6" {
		t.Fatalf("label after +1 = %q", label.Text)
	}
}

func stepperLabel(root fyne.CanvasObject) *widget.Label {
	var found *widget.Label
	walk(root, func(o fyne.CanvasObject) {
		if l, ok := o.(*widget.Label); ok {
			found = l
		}
	})
	return found
}
