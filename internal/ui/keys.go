package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	Tab       key.Binding
	Reset     key.Binding
	ThreshUp  key.Binding
	ThreshDn  key.Binding
	OpenDir   key.Binding
	OpenLog   key.Binding
	Quit      key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch tab"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	ThreshUp: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "threshold +5%"),
	),
	ThreshDn: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "threshold -5%"),
	),
	OpenDir: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open log dir"),
	),
	OpenLog: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "open last log"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
