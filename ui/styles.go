package ui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor = lipgloss.Color("#FF6B6B")
	accentColor  = lipgloss.Color("#FFE66D")
	mutedColor   = lipgloss.Color("#6C6C6C")
	successColor = lipgloss.Color("#6BFF9E")

	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle()

	dimStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	activeInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	inactiveInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor).
				Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)
