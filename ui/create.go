package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createState int

const (
	stateForm createState = iota
	stateLoading
	stateSuccess
	stateError
)

const (
	inputName = iota
	inputDesc
)

type createAppModel struct {
	state      createState
	inputs     []textinput.Model
	focused    int
	visibility string
	errMsg     string
	successMsg string
	spinner    spinner.Model
}

type appCreatedMsg struct{ output string }
type appCreateErrMsg struct{ err error }

func newCreateAppModel() createAppModel {
	name := textinput.New()
	name.Placeholder = "my-awesome-app"
	name.CharLimit = 100
	name.Width = 36
	name.Focus()

	desc := textinput.New()
	desc.Placeholder = "A short description (optional)"
	desc.CharLimit = 255
	desc.Width = 36

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = dimStyle

	return createAppModel{
		state:      stateForm,
		inputs:     []textinput.Model{name, desc},
		focused:    inputName,
		visibility: "public",
		spinner:    s,
	}
}

func createRepo(name, desc, visibility string) tea.Cmd {
	return func() tea.Msg {
		args := []string{
			"repo", "create", name,
			"--" + visibility,
		}
		if desc != "" {
			args = append(args, "--description", desc)
		}
		out, err := exec.Command("gh", args...).CombinedOutput() //nolint:gosec
		if err != nil {
			return appCreateErrMsg{
				err: fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out))),
			}
		}
		return appCreatedMsg{output: strings.TrimSpace(string(out))}
	}
}

func (m createAppModel) Update(msg tea.Msg) (createAppModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case appCreatedMsg:
		m.state = stateSuccess
		m.successMsg = msg.output
		return m, nil

	case appCreateErrMsg:
		m.state = stateError
		m.errMsg = msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		if m.state == stateLoading {
			return m, nil
		}

		switch msg.String() {
		case "esc":
			return newCreateAppModel(), func() tea.Msg { return navigateBackMsg{} }

		case "enter":
			if m.state == stateSuccess || m.state == stateError {
				return newCreateAppModel(), func() tea.Msg { return navigateBackMsg{} }
			}
			name := strings.TrimSpace(m.inputs[inputName].Value())
			if name == "" {
				m.errMsg = "App name is required"
				return m, nil
			}
			m.state = stateLoading
			m.errMsg = ""
			return m, tea.Batch(
				m.spinner.Tick,
				createRepo(name, m.inputs[inputDesc].Value(), m.visibility),
			)

		case "tab":
			if m.state == stateForm {
				if m.focused < len(m.inputs) {
					m.inputs[m.focused].Blur()
				}
				m.focused = (m.focused + 1) % (len(m.inputs) + 1)
				if m.focused < len(m.inputs) {
					m.inputs[m.focused].Focus()
				}
			}

		case "shift+tab":
			if m.state == stateForm {
				if m.focused < len(m.inputs) {
					m.inputs[m.focused].Blur()
				}
				m.focused = (m.focused - 1 + len(m.inputs) + 1) % (len(m.inputs) + 1)
				if m.focused < len(m.inputs) {
					m.inputs[m.focused].Focus()
				}
			}

		case " ":
			if m.state == stateForm && m.focused == len(m.inputs) {
				if m.visibility == "public" {
					m.visibility = "private"
				} else {
					m.visibility = "public"
				}
			} else if m.state == stateForm && m.focused < len(m.inputs) {
				var cmd tea.Cmd
				m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
				return m, cmd
			}

		default:
			if m.state == stateForm && m.focused < len(m.inputs) {
				var cmd tea.Cmd
				m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m createAppModel) View() string {
	switch m.state {
	case stateLoading:
		return m.spinner.View() + " Creating repository on GitHub..."

	case stateSuccess:
		var sb strings.Builder
		sb.WriteString(successStyle.Render("✓ Repository created successfully!"))
		if m.successMsg != "" {
			sb.WriteString("\n\n")
			sb.WriteString(dimStyle.Render(m.successMsg))
		}
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("press enter or esc to go back"))
		return sb.String()

	case stateError:
		var sb strings.Builder
		sb.WriteString(errorStyle.Render("✗ Failed to create repository"))
		sb.WriteString("\n\n")
		sb.WriteString(m.errMsg)
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("press enter or esc to go back"))
		return sb.String()
	}

	// Form
	var sb strings.Builder

	sb.WriteString(inputLabelStyle.Render("App name"))
	sb.WriteString("\n")
	if m.focused == inputName {
		sb.WriteString(activeInputStyle.Render(m.inputs[inputName].View()))
	} else {
		sb.WriteString(inactiveInputStyle.Render(m.inputs[inputName].View()))
	}
	sb.WriteString("\n\n")

	sb.WriteString(inputLabelStyle.Render("Description"))
	sb.WriteString("\n")
	if m.focused == inputDesc {
		sb.WriteString(activeInputStyle.Render(m.inputs[inputDesc].View()))
	} else {
		sb.WriteString(inactiveInputStyle.Render(m.inputs[inputDesc].View()))
	}
	sb.WriteString("\n\n")

	sb.WriteString(inputLabelStyle.Render("Visibility"))
	sb.WriteString("\n")
	var pub, priv string
	if m.visibility == "public" {
		pub = "● public"
		priv = "○ private"
	} else {
		pub = "○ public"
		priv = "● private"
	}
	visContent := pub + "    " + priv
	if m.focused == len(m.inputs) {
		sb.WriteString(activeInputStyle.Render(visContent))
	} else {
		sb.WriteString(inactiveInputStyle.Render(visContent))
	}
	sb.WriteString("\n\n")

	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render("✗ " + m.errMsg))
		sb.WriteString("\n\n")
	}

	sb.WriteString(helpStyle.Render("tab: next field  •  space: toggle visibility  •  enter: create  •  esc: back"))

	return sb.String()
}
