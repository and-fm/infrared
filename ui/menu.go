package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type menuAction int

const (
	actionCreateApp menuAction = iota
	actionConfigureWebhooks
	actionQuit
)

type menuEntry struct {
	label string
	desc  string
	value menuAction
}

type menuModel struct {
	items  []menuEntry
	cursor int
}

func newMenuModel() menuModel {
	return menuModel{
		items: []menuEntry{
			{
				label: "Create new app",
				desc:  "Create a new GitHub repository",
				value: actionCreateApp,
			},
			{
				label: "Configure webhooks",
				desc:  "Set up Flux receiver webhooks on a GitHub repo",
				value: actionConfigureWebhooks,
			},
			{
				label: "Quit",
				desc:  "Exit infrared",
				value: actionQuit,
			},
		},
	}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	var sb strings.Builder
	for i, item := range m.items {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if i == m.cursor {
			sb.WriteString(selectedItemStyle.Render("▶ " + item.label))
		} else {
			sb.WriteString(normalItemStyle.Render("  " + item.label))
		}
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("  " + item.desc))
	}
	return sb.String()
}

func (m menuModel) selected() menuAction {
	return m.items[m.cursor].value
}
