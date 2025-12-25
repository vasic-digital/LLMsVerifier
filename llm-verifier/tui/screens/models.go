package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-verifier/client"
)

type ModelsScreen struct {
	client   *client.Client
	width    int
	height   int
	models   []Model
	selected int
	scroll   int
	filter   string
	loading  bool
}

type Model struct {
	ID           string
	Name         string
	Provider     string
	Score        float64
	Verified     bool
	Capabilities []string
}

func NewModelsScreen(client *client.Client) *ModelsScreen {
	return &ModelsScreen{
		client:   client,
		models:   []Model{},
		selected: 0,
		scroll:   0,
		filter:   "",
		loading:  false,
	}
}

func (m *ModelsScreen) Init() tea.Cmd {
	return tea.Batch(
		m.loadModels(),
		modelsTickCmd(),
	)
}

func modelsTickCmd() tea.Cmd {
	return tea.Tick(time.Second*60, func(t time.Time) tea.Msg {
		return modelsTickMsg(t)
	})
}

type modelsTickMsg time.Time

func (m *ModelsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case modelsTickMsg:
		return m, tea.Batch(m.loadModels(), modelsTickCmd())
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.adjustScroll()
			}
		case "down", "j":
			if m.selected < len(m.filteredModels())-1 {
				m.selected++
				m.adjustScroll()
			}
		case "enter", " ":
			if len(m.filteredModels()) > 0 {
				return m, m.verifyModel(m.filteredModels()[m.selected].ID)
			}
		case "r", "R":
			return m, m.loadModels()
		case "f", "/":
			return m, m.startFilter()
		case "esc":
			m.filter = ""
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= "a" && msg.String() <= "z" || msg.String() >= "A" && msg.String() <= "Z" || msg.String() >= "0" && msg.String() <= "9" || msg.String() == "-" || msg.String() == "_" {
				m.filter += msg.String()
			}
		}
	case ModelsLoadedMsg:
		m.models = msg.Models
		m.loading = false
	case ModelVerifiedMsg:
		for i, model := range m.models {
			if model.ID == msg.ModelID {
				m.models[i].Verified = true
				m.models[i].Score = msg.Score
				break
			}
		}
	case ModelsErrorMsg, ModelErrorMsg:
		// Log error but don't crash
		fmt.Printf("Error: %v\n", msg)
	}

	return m, nil
}

func (m *ModelsScreen) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Render("🤖 Models")

	filterDisplay := ""
	if m.filter != "" {
		filterDisplay = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Render(fmt.Sprintf("Filter: %s", m.filter))
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().Width(m.width-lipgloss.Width(title)-2).Align(lipgloss.Right).Render(filterDisplay),
	)

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		m.renderModelsList(),
		m.renderModelDetails(),
		m.renderActions(),
	)

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(m.width - 4).
		Height(m.height - 6)

	return contentStyle.Render(content)
}

func (m *ModelsScreen) renderModelsList() string {
	filteredModels := m.filteredModels()

	// Summary statistics
	total := len(m.models)
	verified := 0
	for _, model := range m.models {
		if model.Verified {
			verified++
		}
	}

	summary := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Render(fmt.Sprintf("Total: %d", total)),
		lipgloss.NewStyle().Width(3).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Render(fmt.Sprintf("Verified: %d", verified)),
		lipgloss.NewStyle().Width(3).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Render(fmt.Sprintf("Pending: %d", total-verified)),
	)

	if len(filteredModels) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Top,
			summary,
			lipgloss.NewStyle().Height(1).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(1, 0).
				Render("No models found"+m.filterMessage()),
		)
	}

	listHeight := m.height/2 - 8
	visibleModels := filteredModels[m.scroll:min(m.scroll+listHeight, len(filteredModels))]

	var rows []string
	for i, model := range visibleModels {
		index := m.scroll + i
		isSelected := index == m.selected

		rowStyle := lipgloss.NewStyle()
		if isSelected {
			rowStyle = rowStyle.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("255"))
		}

		status := "✓"
		statusColor := "46"
		if !model.Verified {
			status = "●"
			statusColor = "214"
		}

		scoreColor := "46"
		if model.Score < 70 {
			scoreColor = "196"
		} else if model.Score < 85 {
			scoreColor = "214"
		}

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(statusColor)).
				Width(2).
				Render(status),
			lipgloss.NewStyle().Width(1).Render(""),
			lipgloss.NewStyle().
				Bold(true).
				Width(20).
				Render(model.Name),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Width(12).
				Render(model.Provider),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(scoreColor)).
				Width(8).
				Align(lipgloss.Right).
				Render(fmt.Sprintf("%.1f", model.Score)),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(30).
				Render(strings.Join(model.Capabilities, ", ")),
		)

		rows = append(rows, rowStyle.Render(row))
	}

	return lipgloss.NewStyle().
		Padding(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				summary,
				lipgloss.NewStyle().Height(1).Render(""),
				lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Models (%d/%d):", len(filteredModels), len(m.models)))+m.filterMessage(),
				lipgloss.NewStyle().Height(1).Render(""),
				lipgloss.JoinVertical(lipgloss.Top, rows...),
			),
		)
}

func (m *ModelsScreen) renderModelDetails() string {
	if len(m.filteredModels()) == 0 || m.selected >= len(m.filteredModels()) {
		return ""
	}

	model := m.filteredModels()[m.selected]

	details := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Selected Model:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Name:"),
			lipgloss.NewStyle().Bold(true).Render(model.Name),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Provider:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(model.Provider),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Score:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(fmt.Sprintf("%.1f", model.Score)),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Status:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(func() string {
				if model.Verified {
					return "46"
				}
				return "214"
			}())).Render(func() string {
				if model.Verified {
					return "Verified"
				}
				return "Pending"
			}()),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Capabilities:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(strings.Join(model.Capabilities, ", ")),
		),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		MarginTop(1).
		Render(details)
}

func (m *ModelsScreen) renderActions() string {
	actions := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Actions:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderActionButton("↑/↓", "Navigate", "Select model"),
			lipgloss.NewStyle().Width(2).Render(""),
			m.renderActionButton("Enter", "Verify", "Run verification"),
			lipgloss.NewStyle().Width(2).Render(""),
			m.renderActionButton("f", "Filter", "Search models"),
			lipgloss.NewStyle().Width(2).Render(""),
			m.renderActionButton("r", "Refresh", "Reload models"),
		),
	)

	return lipgloss.NewStyle().
		Padding(1, 0).
		MarginTop(1).
		Render(actions)
}

func (m *ModelsScreen) renderActionButton(key, label, description string) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(20).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("39")).
					Render(fmt.Sprintf("[%s] %s", key, label)),
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Render(description),
			),
		)
}

func (m *ModelsScreen) filteredModels() []Model {
	if m.filter == "" {
		return m.models
	}

	filterLower := strings.ToLower(m.filter)
	var filtered []Model
	for _, model := range m.models {
		if strings.Contains(strings.ToLower(model.Name), filterLower) ||
			strings.Contains(strings.ToLower(model.Provider), filterLower) ||
			strings.Contains(strings.ToLower(strings.Join(model.Capabilities, " ")), filterLower) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func (m *ModelsScreen) filterMessage() string {
	if m.filter == "" {
		return ""
	}
	filteredCount := len(m.filteredModels())
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Render(fmt.Sprintf(" (filtered: %d)", filteredCount))
}

func (m *ModelsScreen) adjustScroll() {
	listHeight := m.height/2 - 8

	if m.selected < m.scroll {
		m.scroll = m.selected
	} else if m.selected >= m.scroll+listHeight {
		m.scroll = m.selected - listHeight + 1
	}
}

func (m *ModelsScreen) loadModels() tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		apiModels, err := m.client.GetModels()
		if err != nil {
			return ModelsErrorMsg{Error: err}
		}

		var models []Model
		for _, apiModel := range apiModels {
			model := Model{
				ID:       getString(apiModel, "id"),
				Name:     getString(apiModel, "name"),
				Provider: getString(apiModel, "provider"),
				Score:    getFloat64(apiModel, "score"),
				Verified: getBool(apiModel, "verified"),
			}

			// Parse capabilities
			if caps, ok := apiModel["capabilities"].([]any); ok {
				for _, cap := range caps {
					if capStr, ok := cap.(string); ok {
						model.Capabilities = append(model.Capabilities, capStr)
					}
				}
			}

			models = append(models, model)
		}

		return ModelsLoadedMsg{
			Models: models,
		}
	}
}

func (m *ModelsScreen) startFilter() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return FilterStartedMsg{}
		},
	)
}

func (m *ModelsScreen) verifyModel(modelID string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.VerifyModel(modelID)
		if err != nil {
			return ModelErrorMsg{Error: err}
		}

		score := 0.0
		if s, ok := result["score"].(float64); ok {
			score = s
		}

		return ModelVerifiedMsg{
			ModelID: modelID,
			Score:   score,
		}
	}
}

type ModelsLoadedMsg struct {
	Models []Model
}

type ModelVerifiedMsg struct {
	ModelID string
	Score   float64
}

type ModelErrorMsg struct {
	Error error
}

type FilterStartedMsg struct{}

type ModelsErrorMsg struct {
	Error error
}

func getString(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(m map[string]any, key string) float64 {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0.0
}

func getBool(m map[string]any, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
