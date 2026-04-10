package web

import "github.com/adnlv/gotaro/internal/domain"

// Status and priority colors for badges and form accents (WCAG-friendly pairs).

func statusPresentation(s domain.Status) (label, bg, fg string) {
	switch s {
	case domain.StatusTodo:
		return "To do", "#6c757d", "#ffffff"
	case domain.StatusInProgress:
		return "In progress", "#0d6efd", "#ffffff"
	case domain.StatusDone:
		return "Done", "#198754", "#ffffff"
	default:
		return s.String(), "#6c757d", "#ffffff"
	}
}

func statusPresentationFromSlug(slug string) (label, bg, fg string) {
	if s, ok := domain.StatusFromString(slug); ok {
		return statusPresentation(s)
	}
	if slug == "" {
		return statusPresentation(domain.StatusTodo)
	}
	return slug, "#6c757d", "#ffffff"
}

func priorityPresentation(p domain.Priority) (label, bg, fg string) {
	switch p {
	case domain.PriorityLow:
		return "Low", "#0aa2c0", "#ffffff"
	case domain.PriorityMedium:
		return "Medium", "#ffc107", "#212529"
	case domain.PriorityHigh:
		return "High", "#dc3545", "#ffffff"
	default:
		return p.String(), "#6c757d", "#ffffff"
	}
}

func priorityPresentationFromSlug(slug string) (label, bg, fg string) {
	if p, ok := domain.PriorityFromString(slug); ok {
		return priorityPresentation(p)
	}
	if slug == "" {
		return priorityPresentation(domain.PriorityMedium)
	}
	return slug, "#6c757d", "#ffffff"
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
