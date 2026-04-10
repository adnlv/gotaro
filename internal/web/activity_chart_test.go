package web

import (
	"strings"
	"testing"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
)

func TestActivityChartSVG_empty(t *testing.T) {
	t.Parallel()
	if s := string(activityChartSVG(nil)); s != "" {
		t.Fatalf("expected empty, got %q", s)
	}
}

func TestActivityChartSVG_containsPolylines(t *testing.T) {
	t.Parallel()
	d0 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	points := []domain.DailyActivityPoint{
		{Date: d0, Created: 1, Completed: 0},
		{Date: d0.AddDate(0, 0, 1), Created: 2, Completed: 1},
	}
	s := string(activityChartSVG(points))
	if !strings.Contains(s, `<svg`) || !strings.Contains(s, `stroke="#0d6efd"`) || !strings.Contains(s, `stroke="#198754"`) {
		t.Fatalf("unexpected svg: %s", s)
	}
}
