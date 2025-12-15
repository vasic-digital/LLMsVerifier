package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-verifier/client"
)

type ProvidersScreen struct {
	client    *client.Client
	width     int
	height    int
	providers []Provider
	selected  int
	loading   bool
}

type Provider struct {
	ID         string
	Name       string
	ModelCount int
	AvgScore   float64
	Status     string
	APIKeySet  bool
}

func NewProvidersScreen(client *client.Client) *ProvidersScreen {
	return &ProvidersScreen{
		client:    client,
		providers: []Provider{},
		selected:  0,
		loading:   false,
	}
}

func (p *ProvidersScreen) Init() tea.Cmd {
	return p.loadProviders()
}

func (p *ProvidersScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.selected > 0 {
				p.selected--
			}
		case "down", "j":
			if p.selected < len(p.providers)-1 {
				p.selected++
			}
		case "enter", " ":
			return p, p.toggleProviderStatus(p.providers[p.selected].ID)
		case "a", "A":
			return p, p.addAPIKey(p.providers[p.selected].ID)
		}
	}
	return p, nil
}

func (p *ProvidersScreen) View() string {
	if p.width == 0 || p.height == 0 {
		return "Loading..."
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Render("🏢 Providers")

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		p.renderProvidersList(),
		p.renderProviderDetails(),
		p.renderActions(),
	)

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(p.width - 4).
		Height(p.height - 6)

	return contentStyle.Render(content)
}

func (p *ProvidersScreen) renderProvidersList() string {
	if len(p.providers) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0).
			Render("No providers configured")
	}

	var rows []string
	for i, provider := range p.providers {
		isSelected := i == p.selected

		rowStyle := lipgloss.NewStyle()
		if isSelected {
			rowStyle = rowStyle.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("255"))
		}

		statusColor := "46"
		if provider.Status == "Inactive" {
			statusColor = "196"
		}

		apiKeyStatus := "✓"
		apiKeyColor := "46"
		if !provider.APIKeySet {
			apiKeyStatus = "✗"
			apiKeyColor = "196"
		}

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(statusColor)).
				Width(2).
				Render("●"),
			lipgloss.NewStyle().Width(1).Render(""),
			lipgloss.NewStyle().
				Bold(true).
				Width(15).
				Render(provider.Name),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Width(10).
				Render(fmt.Sprintf("%d models", provider.ModelCount)),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Width(10).
				Align(lipgloss.Right).
				Render(fmt.Sprintf("%.1f", provider.AvgScore)),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(statusColor)).
				Width(10).
				Render(provider.Status),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(apiKeyColor)).
				Width(5).
				Render(apiKeyStatus),
		)

		rows = append(rows, rowStyle.Render(row))
	}

	return lipgloss.NewStyle().
		Padding(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				lipgloss.NewStyle().Bold(true).Render("Providers:"),
				lipgloss.NewStyle().Height(1).Render(""),
				lipgloss.JoinVertical(lipgloss.Top, rows...),
			),
		)
}

func (p *ProvidersScreen) renderProviderDetails() string {
	if len(p.providers) == 0 || p.selected >= len(p.providers) {
		return ""
	}

	provider := p.providers[p.selected]

	details := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Selected Provider:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Name:"),
			lipgloss.NewStyle().Bold(true).Render(provider.Name),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Models:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(fmt.Sprintf("%d", provider.ModelCount)),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Avg Score:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(fmt.Sprintf("%.1f", provider.AvgScore)),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Status:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(func() string {
				if provider.Status == "Active" {
					return "46"
				}
				return "196"
			}())).Render(provider.Status),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("API Key:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(func() string {
				if provider.APIKeySet {
					return "46"
				}
				return "196"
			}())).Render(func() string {
				if provider.APIKeySet {
					return "Configured"
				}
				return "Not configured"
			}()),
		),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		MarginTop(1).
		Render(details)
}

func (p *ProvidersScreen) renderActions() string {
	actions := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Actions:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			p.renderActionButton("↑/↓", "Navigate", "Select provider"),
			lipgloss.NewStyle().Width(2).Render(""),
			p.renderActionButton("Enter", "Toggle", "Toggle status"),
			lipgloss.NewStyle().Width(2).Render(""),
			p.renderActionButton("a", "API Key", "Add API key"),
		),
	)

	return lipgloss.NewStyle().
		Padding(1, 0).
		MarginTop(1).
		Render(actions)
}

func (p *ProvidersScreen) renderActionButton(key, label, description string) string {
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

func (p *ProvidersScreen) toggleProviderStatus(providerID string) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would call the API
		// For now, just simulate the toggle
		return ProviderStatusToggledMsg{
			ProviderID: providerID,
		}
	}
}

func (p *ProvidersScreen) addAPIKey(providerID string) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would call the API
		// For now, just simulate adding API key
		return APIKeyAddedMsg{
			ProviderID: providerID,
		}
	}
}

type ProviderStatusToggledMsg struct {
	ProviderID string
}

type APIKeyAddedMsg struct {
	ProviderID string
}
