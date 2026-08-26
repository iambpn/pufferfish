package window

import (
	"testing"

	"github.com/iambpn/pufferfish/internal/preferences"
)

func TestHistoryWindowOriginAnchorsToTheRequestedCorner(t *testing.T) {
	const screenW, screenH = 1920, 1080

	cases := []struct {
		pos  preferences.HistoryPosition
		x, y int
	}{
		{preferences.HistoryPositionCenter, (screenW - historyWindowWidth) / 2, (screenH - historyWindowHeight) / 2},
		{preferences.HistoryPositionTopLeft, historyWindowMargin, historyWindowMargin},
		{preferences.HistoryPositionTopCenter, (screenW - historyWindowWidth) / 2, historyWindowMargin},
		{preferences.HistoryPositionTopRight, screenW - historyWindowWidth - historyWindowMargin, historyWindowMargin},
		{preferences.HistoryPositionCenterLeft, historyWindowMargin, (screenH - historyWindowHeight) / 2},
		{preferences.HistoryPositionCenterRight, screenW - historyWindowWidth - historyWindowMargin, (screenH - historyWindowHeight) / 2},
		{preferences.HistoryPositionBottomLeft, historyWindowMargin, screenH - historyWindowHeight - historyWindowMargin},
		{preferences.HistoryPositionBottomCenter, (screenW - historyWindowWidth) / 2, screenH - historyWindowHeight - historyWindowMargin},
		{preferences.HistoryPositionBottomRight, screenW - historyWindowWidth - historyWindowMargin, screenH - historyWindowHeight - historyWindowMargin},
	}

	for _, c := range cases {
		x, y := historyWindowOrigin(c.pos, screenW, screenH)
		if x != c.x || y != c.y {
			t.Errorf("%s: got (%d, %d), want (%d, %d)", c.pos, x, y, c.x, c.y)
		}
	}
}

func TestHistoryWindowOriginFallsBackForAnUnknownPosition(t *testing.T) {
	x, y := historyWindowOrigin("bogus", 1920, 1080)
	if x != historyWindowFallbackX || y != historyWindowFallbackY {
		t.Fatalf("got (%d, %d)", x, y)
	}
}
