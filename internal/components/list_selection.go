package components

import (
	"strings"
)

func NewListSelector(items []string) *ListSelector {
	return &ListSelector{
		items:  items,
		Cursor: NewCursor(len(items) - 1),
	}
}

type ListSelector struct {
	items  []string
	Cursor *Cursor
}

func (s *ListSelector) Render() string {
	render := strings.Builder{}

	for i, item := range s.items {
		if i == s.Cursor.Cursor() {
			render.WriteString("> " + item)
		} else {
			render.WriteString(item)
		}

		render.WriteString(LineBreak)
	}

	return render.String()
}
