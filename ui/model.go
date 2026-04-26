package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type viewType int

const (
	viewMenu viewType = iota
	viewCreateApp
	viewConfigureWebhooks
)

type navigateBackMsg struct{}

// Model is the root application model.
type Model struct {
	currentView viewType
	menu        menuModel
	createApp   createAppModel
	webhook     webhookModel
}

// New creates and returns the root application model.
func New() Model {
	return Model{
		currentView: viewMenu,
		menu:        newMenuModel(),
		createApp:   newCreateAppModel(),
		webhook:     newWebhookModel(),
	}
}

func (m Model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case navigateBackMsg:
		m.currentView = viewMenu
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.currentView == viewMenu {
				return m, tea.Quit
			}
		case "enter":
			if m.currentView == viewMenu {
				switch m.menu.selected() {
				case actionCreateApp:
					m.currentView = viewCreateApp
					m.createApp = newCreateAppModel()
					return m, nil
				case actionConfigureWebhooks:
					m.currentView = viewConfigureWebhooks
					m.webhook = newWebhookModel()
					return m, nil
				case actionQuit:
					return m, tea.Quit
				}
			}
		}
	}

	// Dispatch to the active sub-model.
	switch m.currentView {
	case viewMenu:
		m.menu, cmd = m.menu.Update(msg)
	case viewCreateApp:
		m.createApp, cmd = m.createApp.Update(msg)
	case viewConfigureWebhooks:
		m.webhook, cmd = m.webhook.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	var sb strings.Builder

	switch m.currentView {
	case viewMenu:
		sb.WriteString(titleStyle.Render("infrared"))
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render("infra management made easy"))
		sb.WriteString("\n\n")
		sb.WriteString(m.menu.View())
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("↑/↓ or j/k: navigate  •  enter: select  •  q: quit"))

	case viewCreateApp:
		sb.WriteString(titleStyle.Render("infrared"))
		sb.WriteString(dimStyle.Render(" / "))
		sb.WriteString(titleStyle.Render("new app"))
		sb.WriteString("\n\n")
		sb.WriteString(m.createApp.View())

	case viewConfigureWebhooks:
		sb.WriteString(titleStyle.Render("infrared"))
		sb.WriteString(dimStyle.Render(" / "))
		sb.WriteString(titleStyle.Render("configure webhooks"))
		sb.WriteString("\n\n")
		sb.WriteString(m.webhook.View())
	}

	return appStyle.Render(sb.String())
}
