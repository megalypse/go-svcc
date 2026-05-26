package components

import tea "github.com/charmbracelet/bubbletea"

func NewCursor(limit int) *Cursor {
	return &Cursor{
		limit: limit,
	}
}

type Cursor struct {
	cursor int
	limit  int
}

func (c *Cursor) Up() {
	if c.cursor < c.limit {
		c.cursor++
	}
}

func (c *Cursor) Down() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *Cursor) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			c.Down()
		case "down", "j":
			c.Up()
		}
	}
}

func (c *Cursor) Cursor() int {
	return c.cursor
}
