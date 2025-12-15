package screens

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-verifier/client"
)

type VerificationScreen struct {
	client          *client.Client
	width           int
	height          int
	verifications   []Verification
	selected        int
	showNewForm     bool
	newVerification NewVerification
}

type Verification struct {
	ID          int
	ModelName   string
	Provider    string
	Status      string
	Score       float64
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
}

type NewVerification struct {
	ModelID   int
	Provider  string
	TestTypes []string
}

func NewVerificationScreen(client *client.Client) *VerificationScreen {
	return &VerificationScreen{
		client:        client,
		verifications: []Verification{}, // Will be populated with real data
		selected:      0,
		showNewForm:   false,
		newVerification: NewVerification{
			ModelID:   0,
			Provider:  "",
			TestTypes: []string{"code", "reasoning", "tools"},
		},
	}
}

func (v *VerificationScreen) Init() tea.Cmd {
	return nil
}

func (v *VerificationScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if v.selected > 0 {
				v.selected--
			}
		case "down", "j":
			if v.selected < len(v.verifications)-1 {
				v.selected++
			}
		case "n", "N":
			v.showNewForm = !v.showNewForm
		case "enter", " ":
			if v.showNewForm {
				return v, v.startNewVerification()
			} else if len(v.verifications) > 0 {
				return v, v.cancelVerification(v.verifications[v.selected].ID)
			}
		case "r", "R":
			return v, v.refreshVerifications()
		}
	}
	return v, nil
}

func (v *VerificationScreen) View() string {
	if v.width == 0 || v.height == 0 {
		return "Loading..."
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Render("🔍 Verification")

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		v.renderVerificationsList(),
		v.renderVerificationDetails(),
		v.renderActions(),
	)

	if v.showNewForm {
		content = lipgloss.JoinVertical(
			lipgloss.Top,
			content,
			v.renderNewVerificationForm(),
		)
	}

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(v.width - 4).
		Height(v.height - 6)

	return contentStyle.Render(content)
}

func (v *VerificationScreen) renderVerificationsList() string {
	if len(v.verifications) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0).
			Render("No verification history")
	}

	var rows []string
	for i, verification := range v.verifications {
		isSelected := i == v.selected

		rowStyle := lipgloss.NewStyle()
		if isSelected {
			rowStyle = rowStyle.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("255"))
		}

		statusColor := "46"
		switch verification.Status {
		case "Running":
			statusColor = "214"
		case "Failed":
			statusColor = "196"
		case "Pending":
			statusColor = "240"
		}

		statusIcon := "✓"
		switch verification.Status {
		case "Running":
			statusIcon = "⟳"
		case "Failed":
			statusIcon = "✗"
		case "Pending":
			statusIcon = "●"
		}

		scoreDisplay := fmt.Sprintf("%.1f", verification.Score)
		if verification.Score == 0 {
			scoreDisplay = "-"
		}

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(statusColor)).
				Width(2).
				Render(statusIcon),
			lipgloss.NewStyle().Width(1).Render(""),
			lipgloss.NewStyle().
				Bold(true).
				Width(20).
				Render(verification.ModelName),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Width(12).
				Render(verification.Provider),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(statusColor)).
				Width(12).
				Render(verification.Status),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Width(8).
				Align(lipgloss.Right).
				Render(scoreDisplay),
			lipgloss.NewStyle().Width(2).Render(""),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(15).
				Render(v.formatDuration(verification.Duration)),
		)

		rows = append(rows, rowStyle.Render(row))
	}

	return lipgloss.NewStyle().
		Padding(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				lipgloss.NewStyle().Bold(true).Render("Verification History:"),
				lipgloss.NewStyle().Height(1).Render(""),
				lipgloss.JoinVertical(lipgloss.Top, rows...),
			),
		)
}

func (v *VerificationScreen) renderVerificationDetails() string {
	if len(v.verifications) == 0 || v.selected >= len(v.verifications) {
		return ""
	}

	verification := v.verifications[v.selected]

	details := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Selected Verification:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Model:"),
			lipgloss.NewStyle().Bold(true).Render(verification.ModelName),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Provider:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(verification.Provider),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Status:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(func() string {
				switch verification.Status {
				case "Completed":
					return "46"
				case "Running":
					return "214"
				case "Failed":
					return "196"
				default:
					return "240"
				}
			}())).Render(verification.Status),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Score:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(fmt.Sprintf("%.1f", verification.Score)),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Started:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(verification.StartedAt.Format("2006-01-02 15:04:05")),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(20).Render("Duration:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(v.formatDuration(verification.Duration)),
		),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		MarginTop(1).
		Render(details)
}

func (v *VerificationScreen) renderActions() string {
	actions := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Actions:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderActionButton("↑/↓", "Navigate", "Select verification"),
			lipgloss.NewStyle().Width(2).Render(""),
			v.renderActionButton("n", "New", "Start new verification"),
			lipgloss.NewStyle().Width(2).Render(""),
			v.renderActionButton("Enter", "Cancel", "Cancel selected"),
			lipgloss.NewStyle().Width(2).Render(""),
			v.renderActionButton("r", "Refresh", "Update status"),
		),
	)

	return lipgloss.NewStyle().
		Padding(1, 0).
		MarginTop(1).
		Render(actions)
}

func (v *VerificationScreen) renderActionButton(key, label, description string) string {
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

func (v *VerificationScreen) renderNewVerificationForm() string {
	form := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Render("New Verification:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Model ID:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(fmt.Sprintf("%d", v.newVerification.ModelID)),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Provider:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(v.newVerification.Provider),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render("Test Types:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(fmt.Sprintf("%v", v.newVerification.TestTypes)),
		),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Press Enter to start verification, 'n' to close"),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1).
		MarginTop(1).
		Render(form)
}

func (v *VerificationScreen) formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func (v *VerificationScreen) startNewVerification() tea.Cmd {
	return func() tea.Msg {
		return NewVerificationStartedMsg{
			Verification: v.newVerification,
		}
	}
}

func (v *VerificationScreen) cancelVerification(verificationID int) tea.Cmd {
	return func() tea.Msg {
		return VerificationCancelledMsg{
			VerificationID: verificationID,
		}
	}
}

func (v *VerificationScreen) refreshVerifications() tea.Cmd {
	return func() tea.Msg {
		return VerificationsRefreshedMsg{}
	}
}

type NewVerificationStartedMsg struct {
	Verification NewVerification
}

type VerificationCancelledMsg struct {
	VerificationID int
}

type VerificationsRefreshedMsg struct{}
