package web

import "github.com/adnlv/gotaro/internal/domain"

// Status and priority colors for badges and form accents (WCAG-friendly pairs).

func statusPresentation(s domain.Status) (label, bg, fg string) {
	switch s {
	case domain.StatusTodo:
		return "To do", "#64748b", "#ffffff"
	case domain.StatusInProgress:
		return "In progress", "#3b82f6", "#ffffff"
	case domain.StatusDone:
		return "Done", "#22c55e", "#ffffff"
	default:
		return s.String(), "#64748b", "#ffffff"
	}
}

func statusPresentationFromSlug(slug string) (label, bg, fg string) {
	if s, ok := domain.StatusFromString(slug); ok {
		return statusPresentation(s)
	}
	if slug == "" {
		return statusPresentation(domain.StatusTodo)
	}
	return slug, "#64748b", "#ffffff"
}

func priorityPresentation(p domain.Priority) (label, bg, fg string) {
	switch p {
	case domain.PriorityLow:
		return "Low", "#06b6d4", "#ffffff"
	case domain.PriorityMedium:
		return "Medium", "#f59e0b", "#1c1917"
	case domain.PriorityHigh:
		return "High", "#ef4444", "#ffffff"
	default:
		return p.String(), "#64748b", "#ffffff"
	}
}

func priorityPresentationFromSlug(slug string) (label, bg, fg string) {
	if p, ok := domain.PriorityFromString(slug); ok {
		return priorityPresentation(p)
	}
	if slug == "" {
		return priorityPresentation(domain.PriorityMedium)
	}
	return slug, "#64748b", "#ffffff"
}

func decorateTaskFormColors(fv *TaskFormView) {
	sl, sbg, sfg := statusPresentationFromSlug(fv.Status)
	pl, pbg, pfg := priorityPresentationFromSlug(fv.Priority)
	fv.StatusLabel = sl
	fv.StatusBG = sbg
	fv.StatusFG = sfg
	fv.PriorityLabel = pl
	fv.PriorityBG = pbg
	fv.PriorityFG = pfg
}
