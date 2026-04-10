package web

import (
	"fmt"
	"html"
	"html/template"
	"strings"

	"github.com/adnlv/gotaro/internal/domain"
)

// taskActivityChartDays is how many UTC calendar days (through today) the summary chart covers.
const taskActivityChartDays = 14

// activityChartSVG draws a dual-line chart (created vs completed per UTC day). Safe for embedding: values are numeric; date labels are escaped.
func activityChartSVG(points []domain.DailyActivityPoint) template.HTML {
	if len(points) == 0 {
		return ""
	}

	maxY := 0
	for _, p := range points {
		if p.Created > maxY {
			maxY = p.Created
		}
		if p.Completed > maxY {
			maxY = p.Completed
		}
	}
	if maxY < 1 {
		maxY = 1
	}

	const (
		W, H = 560, 140
		padL = 36
		padR = 12
		padT = 30
		padB = 24
	)
	iw := float64(W - padL - padR)
	ih := float64(H - padT - padB)
	n := len(points)

	fy := func(v int) float64 {
		return float64(padT) + ih*(1.0-float64(v)/float64(maxY))
	}
	fx := func(i int) float64 {
		if n <= 1 {
			return float64(padL) + iw/2
		}
		return float64(padL) + iw*float64(i)/float64(n-1)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="task-activity-svg" viewBox="0 0 %d %d" width="100%%" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Tasks created and completed per UTC day">`, W, H)

	fmt.Fprintf(&b, `<text x="%d" y="18" fill="#495057" font-size="12" font-weight="600">Last %d days (UTC)</text>`, padL, n)

	yBottom := int(fy(0)) + 1
	yTop := int(fy(maxY))
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#dee2e6" stroke-width="1"/>`, padL, yBottom, padL+int(iw), yBottom)
	if maxY >= 2 {
		mid := maxY / 2
		ym := int(fy(mid))
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#e9ecef" stroke-width="1" stroke-dasharray="5 4"/>`, padL, ym, padL+int(iw), ym)
	}
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#dee2e6" stroke-width="1"/>`, padL, yTop, padL+int(iw), yTop)
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#adb5bd" font-size="10" text-anchor="end">%d</text>`, padL-6, yTop+4, maxY)

	writePoly := func(vals []int, stroke string) {
		var pts strings.Builder
		for i, v := range vals {
			if i > 0 {
				pts.WriteByte(' ')
			}
			fmt.Fprintf(&pts, "%.1f,%.1f", fx(i), fy(v))
		}
		fmt.Fprintf(&b, `<polyline fill="none" stroke="%s" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round" points="%s"/>`, stroke, pts.String())
	}

	created := make([]int, n)
	done := make([]int, n)
	for i, p := range points {
		created[i] = p.Created
		done[i] = p.Completed
	}
	writePoly(created, "#3b82f6")
	writePoly(done, "#22c55e")

	l0 := points[0].Date.Format("Jan 2")
	l1 := points[n-1].Date.Format("Jan 2")
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#868e96" font-size="10">%s</text>`, padL, H-6, html.EscapeString(l0))
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#868e96" font-size="10" text-anchor="end">%s</text>`, W-padR, H-6, html.EscapeString(l1))

	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}
