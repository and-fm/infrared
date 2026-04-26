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
	state       webhookFormState
	appInput    textinput.Model
	secretInput textinput.Model
	envOptions  []webhookEnvOption
	envCursor   int
	focused     int // 0 = appInput, 1 = secretInput, 2 = env list
	results     []webhookEnvResult
	errMsg      string
	spinner     spinner.Model
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

	secret := textinput.New()
	secret.Placeholder = "webhook secret"
	secret.CharLimit = 255
	secret.Width = 36
	secret.EchoMode = textinput.EchoPassword
	secret.EchoCharacter = '•'

	opts := make([]webhookEnvOption, len(allEnvs))
	for i, e := range allEnvs {
		opts[i] = webhookEnvOption{name: e}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = dimStyle

	return webhookModel{
		state:       webhookStateForm,
		appInput:    input,
		secretInput: secret,
		envOptions:  opts,
		spinner:     s,
	}
}

// ---- commands --------------------------------------------------------------

func runWebhookConfig(app, secret string, envs []string) tea.Cmd {
	return func() tea.Msg {
		results := make([]webhookEnvResult, 0, len(envs)+1)
		for _, env := range envs {
			results = append(results, configureEnvWebhook(app, env, secret))
		}
		results = append(results, configureImageWebhook(app, secret))
		return webhookDoneMsg{results: results}
	}
}

func getReceiverURL(receiverName string) (string, error) {
	out, err := exec.Command(
		"kubectl", "get", "receivers", receiverName,
		"-n", "flux-system",
		"-o", "json",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl: %s", strings.TrimSpace(string(out)))
	}

	var resource receiverResource
	if parseErr := json.Unmarshal(out, &resource); parseErr != nil {
		return "", fmt.Errorf("failed to parse receiver output")
	}

	if len(resource.Status.Conditions) == 0 {
		return "", fmt.Errorf("receiver has no status conditions")
	}
	if !strings.EqualFold(resource.Status.Conditions[0].Type, "ready") {
		return "", fmt.Errorf("receiver not ready (condition: %q)", resource.Status.Conditions[0].Type)
	}
	if resource.Status.WebhookPath == "" {
		return "", fmt.Errorf("receiver has empty webhookPath")
	}
	return fluxHookBase + resource.Status.WebhookPath, nil
}

func createGitHubWebhook(app, webhookURL, secret string) error {
	ghArgs := []string{
		"api",
		fmt.Sprintf("repos/%s/%s/hooks", githubOrg, app),
		"-X", "POST",
		"-f", "config[url]=" + webhookURL,
		"-f", "config[content_type]=form",
		"-f", "config[insecure_ssl]=0",
		"-f", "events[]=push",
		"-F", "active=true",
	}
	if secret != "" {
		ghArgs = append(ghArgs, "-f", "config[secret]="+secret)
	}
	out, err := exec.Command("gh", ghArgs...).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("gh api: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func configureEnvWebhook(app, env, secret string) webhookEnvResult {
	r := webhookEnvResult{env: env}
	url, err := getReceiverURL(app + "-repo-receiver-" + env)
	if err != nil {
		r.err = err.Error()
		return r
	}
	r.url = url
	if err := createGitHubWebhook(app, url, secret); err != nil {
		r.err = err.Error()
	}
	return r
}

func configureImageWebhook(app, secret string) webhookEnvResult {
	r := webhookEnvResult{env: "image"}
	url, err := getReceiverURL(app + "-image-receiver")
	if err != nil {
		r.err = err.Error()
		return r
	}
	r.url = url
	if err := createGitHubWebhook(app, url, secret); err != nil {
		r.err = err.Error()
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
				switch m.focused {
				case 0:
					m.appInput.Blur()
					m.secretInput.Focus()
					m.focused = 1
				case 1:
					m.secretInput.Blur()
					m.focused = 2
				case 2:
					m.focused = 0
					m.appInput.Focus()
				}

			case "shift+tab":
				switch m.focused {
				case 0:
					m.appInput.Blur()
					m.focused = 2
				case 1:
					m.secretInput.Blur()
					m.focused = 0
					m.appInput.Focus()
				case 2:
					m.focused = 1
					m.secretInput.Focus()
				}

			case "up", "k":
				if m.focused == 2 && m.envCursor > 0 {
					m.envCursor--
				}

			case "down", "j":
				if m.focused == 2 && m.envCursor < len(m.envOptions)-1 {
					m.envCursor++
				}

			case " ":
				if m.focused == 2 {
					m.envOptions[m.envCursor].selected = !m.envOptions[m.envCursor].selected
				} else if m.focused == 0 {
					var cmd tea.Cmd
					m.appInput, cmd = m.appInput.Update(msg)
					return m, cmd
				} else {
					var cmd tea.Cmd
					m.secretInput, cmd = m.secretInput.Update(msg)
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
					runWebhookConfig(appName, m.secretInput.Value(), selectedEnvs),
				)

			default:
				if m.focused == 0 {
					var cmd tea.Cmd
					m.appInput, cmd = m.appInput.Update(msg)
					return m, cmd
				} else if m.focused == 1 {
					var cmd tea.Cmd
					m.secretInput, cmd = m.secretInput.Update(msg)
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

	sb.WriteString(inputLabelStyle.Render("Webhook secret"))
	sb.WriteString("\n")
	if m.focused == 1 {
		sb.WriteString(activeInputStyle.Render(m.secretInput.View()))
	} else {
		sb.WriteString(inactiveInputStyle.Render(m.secretInput.View()))
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
		if m.focused == 2 && i == m.envCursor {
			envLines.WriteString(selectedItemStyle.Render("▶ " + check + " " + opt.name))
		} else {
			envLines.WriteString(normalItemStyle.Render("  " + check + " " + opt.name))
		}
	}
	if m.focused == 2 {
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
