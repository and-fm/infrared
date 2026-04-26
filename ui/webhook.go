package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const fluxHookBase = "https://fluxhook.and.fm"
const githubOrg = "and-fm"

var allEnvs = []string{"preview", "dev", "prod"}

// ---- state machine --------------------------------------------------------

type webhookFormState int

const (
	webhookStateForm webhookFormState = iota
	webhookStateLoading
	webhookStateResults
)

// ---- data structures -------------------------------------------------------

type webhookEnvOption struct {
	name     string
	selected bool
}

type webhookEnvResult struct {
	env string
	url string
	err string
}

// ---- kubectl JSON shapes ---------------------------------------------------

type receiverResource struct {
	Status struct {
		Conditions []struct {
			Type string `json:"type"`
		} `json:"conditions"`
		WebhookPath string `json:"webhookPath"`
	} `json:"status"`
}

// ---- model -----------------------------------------------------------------

type webhookModel struct {
	state      webhookFormState
	appInput   textinput.Model
	envOptions []webhookEnvOption
	envCursor  int
	focused    int // 0 = appInput, 1 = env list
	results    []webhookEnvResult
	errMsg     string
	spinner    spinner.Model
}

// ---- messages --------------------------------------------------------------

type webhookDoneMsg struct{ results []webhookEnvResult }

// ---- constructor -----------------------------------------------------------

func newWebhookModel() webhookModel {
	input := textinput.New()
	input.Placeholder = "my-app"
	input.CharLimit = 100
	input.Width = 36
	input.Focus()

	opts := make([]webhookEnvOption, len(allEnvs))
	for i, e := range allEnvs {
		opts[i] = webhookEnvOption{name: e}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = dimStyle

	return webhookModel{
		state:      webhookStateForm,
		appInput:   input,
		envOptions: opts,
		spinner:    s,
	}
}

// ---- commands --------------------------------------------------------------

func runWebhookConfig(app string, envs []string) tea.Cmd {
	return func() tea.Msg {
		results := make([]webhookEnvResult, 0, len(envs))
		for _, env := range envs {
			results = append(results, configureEnvWebhook(app, env))
		}
		return webhookDoneMsg{results: results}
	}
}

func configureEnvWebhook(app, env string) webhookEnvResult {
	r := webhookEnvResult{env: env}

	receiverName := app + "-repo-receiver-" + env
	out, err := exec.Command(
		"kubectl", "get", "receivers", receiverName,
		"-n", "flux-system",
		"-o", "json",
	).CombinedOutput()
	if err != nil {
		r.err = "kubectl: " + strings.TrimSpace(string(out))
		return r
	}

	var resource receiverResource
	if parseErr := json.Unmarshal(out, &resource); parseErr != nil {
		r.err = "failed to parse receiver output"
		return r
	}

	if len(resource.Status.Conditions) == 0 {
		r.err = "receiver has no status conditions"
		return r
	}
	if !strings.EqualFold(resource.Status.Conditions[0].Type, "ready") {
		r.err = fmt.Sprintf("receiver not ready (condition: %q)", resource.Status.Conditions[0].Type)
		return r
	}

	if resource.Status.WebhookPath == "" {
		r.err = "receiver has empty webhookPath"
		return r
	}
	r.url = fluxHookBase + resource.Status.WebhookPath

	// Create the GitHub push webhook for this receiver URL.
	ghOut, ghErr := exec.Command(
		"gh", "api",
		fmt.Sprintf("repos/%s/%s/hooks", githubOrg, app),
		"-X", "POST",
		"-f", "config[url]="+r.url,
		"-f", "config[content_type]=json",
		"-f", "config[insecure_ssl]=0",
		"-f", "events[]=push",
		"-F", "active=true",
	).CombinedOutput()
	if ghErr != nil {
		r.err = "gh api: " + strings.TrimSpace(string(ghOut))
		return r
	}

	return r
}

// ---- update ----------------------------------------------------------------

func (m webhookModel) Update(msg tea.Msg) (webhookModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.state == webhookStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case webhookDoneMsg:
		m.state = webhookStateResults
		m.results = msg.results
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case webhookStateLoading:
			return m, nil

		case webhookStateResults:
			switch msg.String() {
			case "enter", "esc":
				return newWebhookModel(), func() tea.Msg { return navigateBackMsg{} }
			}

		case webhookStateForm:
			switch msg.String() {
			case "esc":
				return newWebhookModel(), func() tea.Msg { return navigateBackMsg{} }

			case "tab":
				if m.focused == 0 {
					m.appInput.Blur()
					m.focused = 1
				} else {
					m.focused = 0
					m.appInput.Focus()
				}

			case "shift+tab":
				if m.focused == 1 {
					m.focused = 0
					m.appInput.Focus()
				} else {
					m.appInput.Blur()
					m.focused = 1
				}

			case "up", "k":
				if m.focused == 1 && m.envCursor > 0 {
					m.envCursor--
				}

			case "down", "j":
				if m.focused == 1 && m.envCursor < len(m.envOptions)-1 {
					m.envCursor++
				}

			case " ":
				if m.focused == 1 {
					m.envOptions[m.envCursor].selected = !m.envOptions[m.envCursor].selected
				} else {
					var cmd tea.Cmd
					m.appInput, cmd = m.appInput.Update(msg)
					return m, cmd
				}

			case "enter":
				appName := strings.TrimSpace(m.appInput.Value())
				if appName == "" {
					m.errMsg = "App name is required"
					return m, nil
				}
				var selectedEnvs []string
				for _, opt := range m.envOptions {
					if opt.selected {
						selectedEnvs = append(selectedEnvs, opt.name)
					}
				}
				if len(selectedEnvs) == 0 {
					m.errMsg = "Select at least one environment"
					return m, nil
				}
				m.state = webhookStateLoading
				m.errMsg = ""
				return m, tea.Batch(
					m.spinner.Tick,
					runWebhookConfig(appName, selectedEnvs),
				)

			default:
				if m.focused == 0 {
					var cmd tea.Cmd
					m.appInput, cmd = m.appInput.Update(msg)
					return m, cmd
				}
			}
		}
	}

	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m webhookModel) View() string {
	switch m.state {
	case webhookStateLoading:
		return m.spinner.View() + " Configuring webhooks..."

	case webhookStateResults:
		var sb strings.Builder
		allOK := true
		for i, r := range m.results {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			if r.err != "" {
				allOK = false
				sb.WriteString(errorStyle.Render("✗ " + r.env))
				sb.WriteString("\n  ")
				sb.WriteString(dimStyle.Render(r.err))
			} else {
				sb.WriteString(successStyle.Render("✓ " + r.env))
				sb.WriteString("\n  ")
				sb.WriteString(dimStyle.Render(r.url))
			}
		}
		sb.WriteString("\n\n")
		if allOK {
			sb.WriteString(successStyle.Render("All webhooks configured successfully"))
		} else {
			sb.WriteString(errorStyle.Render("Some webhooks failed — check output above"))
		}
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("press enter or esc to go back"))
		return sb.String()
	}

	// Form view
	var sb strings.Builder

	sb.WriteString(inputLabelStyle.Render("App name"))
	sb.WriteString("\n")
	if m.focused == 0 {
		sb.WriteString(activeInputStyle.Render(m.appInput.View()))
	} else {
		sb.WriteString(inactiveInputStyle.Render(m.appInput.View()))
	}
	sb.WriteString("\n\n")

	sb.WriteString(inputLabelStyle.Render("Environments"))
	sb.WriteString("\n")

	var envLines strings.Builder
	for i, opt := range m.envOptions {
		if i > 0 {
			envLines.WriteString("\n")
		}
		check := "[ ]"
		if opt.selected {
			check = "[✓]"
		}
		if m.focused == 1 && i == m.envCursor {
			envLines.WriteString(selectedItemStyle.Render("▶ " + check + " " + opt.name))
		} else {
			envLines.WriteString(normalItemStyle.Render("  " + check + " " + opt.name))
		}
	}
	if m.focused == 1 {
		sb.WriteString(activeInputStyle.Render(envLines.String()))
	} else {
		sb.WriteString(inactiveInputStyle.Render(envLines.String()))
	}
	sb.WriteString("\n\n")

	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render("✗ " + m.errMsg))
		sb.WriteString("\n\n")
	}

	sb.WriteString(helpStyle.Render("tab: switch field  •  ↑/↓: move  •  space: toggle  •  enter: configure  •  esc: back"))

	return sb.String()
}
