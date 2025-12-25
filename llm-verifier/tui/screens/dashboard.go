package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-verifier/client"
)

type tickMsg time.Time

type DashboardScreen struct {
	client *client.Client
	width  int
	height int
	stats  DashboardStats
}

type DashboardStats struct {
	TotalModels      int
	TotalProviders   int
	VerifiedModels   int
	PendingModels    int
	LastVerification time.Time
	AverageScore     float64
	LastRefresh      time.Time
}

func NewDashboardScreen(client *client.Client) *DashboardScreen {
	return &DashboardScreen{
		client: client,
		stats: DashboardStats{
			TotalModels:      0,
			TotalProviders:   0,
			VerifiedModels:   0,
			PendingModels:    0,
			LastVerification: time.Now(),
			AverageScore:     0.0,
		},
	}
}

func (d *DashboardScreen) Init() tea.Cmd {
	return tea.Batch(
		d.refreshStats(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (d *DashboardScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			return d, d.refreshStats()
		}
	case tickMsg:
		return d, tea.Batch(d.refreshStats(), tickCmd())
	case StatsRefreshedMsg:
		d.stats = msg.Stats
	case StatsErrorMsg:
		// Log error but don't crash
		fmt.Printf("Error refreshing stats: %v\n", msg.Error)
	}
	return d, nil
}

func (d *DashboardScreen) View() string {
	if d.width == 0 || d.height == 0 {
		return "Loading..."
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Render("📊 Dashboard")

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		d.renderStats(),
		d.renderActions(),
	)

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(d.width - 4).
		Height(d.height - 6)

	return contentStyle.Render(content)
}

func (d *DashboardScreen) renderStats() string {
	statsStyle := lipgloss.NewStyle().
		Padding(1, 0)

	statBox := func(label string, value any, color string) string {
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(color)).
			Padding(1, 2).
			Width(20).
			Align(lipgloss.Center).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Center,
					lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color(color)).
						Render(label),
					lipgloss.NewStyle().
						Bold(true).
						Render(fmt.Sprintf("%v", value)),
				),
			)
	}

	stats := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statBox("Total Models", d.stats.TotalModels, "39"),
		lipgloss.NewStyle().Width(2).Render(""),
		statBox("Total Providers", d.stats.TotalProviders, "82"),
		lipgloss.NewStyle().Width(2).Render(""),
		statBox("Verified", d.stats.VerifiedModels, "46"),
		lipgloss.NewStyle().Width(2).Render(""),
		statBox("Pending", d.stats.PendingModels, "214"),
	)

	additionalStats := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statBox("Avg Score", fmt.Sprintf("%.1f%%", d.stats.AverageScore), "205"),
		lipgloss.NewStyle().Width(2).Render(""),
		statBox("Last Verified", d.stats.LastVerification.Format("Jan 02 15:04"), "99"),
		lipgloss.NewStyle().Width(2).Render(""),
		statBox("Last Refresh", d.stats.LastRefresh.Format("15:04:05"), "39"),
	)

	// Add progress visualization
	progressSection := d.renderProgress()

	return statsStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			stats,
			lipgloss.NewStyle().Height(1).Render(""),
			additionalStats,
			lipgloss.NewStyle().Height(1).Render(""),
			progressSection,
		),
	)
}

func (d *DashboardScreen) renderProgress() string {
	if d.stats.TotalModels == 0 {
		return ""
	}

	// Verification progress
	verifiedPercent := float64(d.stats.VerifiedModels) / float64(d.stats.TotalModels) * 100
	progressBar := d.renderProgressBar("Verification Progress", verifiedPercent, 40)

	// Score visualization (simple text-based)
	scoreIndicator := d.renderScoreIndicator()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("📊 Progress Overview:"),
		lipgloss.NewStyle().Height(1).Render(""),
		progressBar,
		lipgloss.NewStyle().Height(1).Render(""),
		scoreIndicator,
	)
}

func (d *DashboardScreen) renderProgressBar(label string, percent float64, width int) string {
	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Width(20).Render(label+":"),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Render(fmt.Sprintf("%.1f%%", percent)),
		lipgloss.NewStyle().Width(2).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Render("["),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Render(bar),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Render("]"),
	)
}

func (d *DashboardScreen) renderScoreIndicator() string {
	score := d.stats.AverageScore
	var color, label string

	if score >= 90 {
		color = "46" // Green
		label = "Excellent"
	} else if score >= 80 {
		color = "226" // Yellow
		label = "Good"
	} else if score >= 70 {
		color = "214" // Orange
		label = "Fair"
	} else {
		color = "196" // Red
		label = "Needs Improvement"
	}

	stars := ""
	for i := range 5 {
		if float64(i) < score/20 {
			stars += "★"
		} else {
			stars += "☆"
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Width(20).Render("Average Score:"),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Bold(true).
			Render(fmt.Sprintf("%.1f", score)),
		lipgloss.NewStyle().Width(2).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Render(stars),
		lipgloss.NewStyle().Width(2).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Render("("+label+")"),
	)
}

func (d *DashboardScreen) renderActions() string {
	actionStyle := lipgloss.NewStyle().
		Padding(1, 0)

	actionButton := func(label, key, description string) string {
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Width(30).
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

	actions := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Quick Actions:"),
		lipgloss.NewStyle().Height(1).Render(""),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			actionButton("Refresh Stats", "r", "Update dashboard statistics (auto-refresh every 30s)"),
			lipgloss.NewStyle().Width(2).Render(""),
			actionButton("Run Verification", "v", "Start new verification"),
			lipgloss.NewStyle().Width(2).Render(""),
			actionButton("View Models", "m", "Browse all models"),
		),
	)

	return actionStyle.Render(actions)
}

func (d *DashboardScreen) refreshStats() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			// Fetch real data from API
			models, err := d.client.GetModels()
			if err != nil {
				return StatsErrorMsg{Error: err}
			}

			providers, err := d.client.GetProviders()
			if err != nil {
				return StatsErrorMsg{Error: err}
			}

			results, err := d.client.GetVerificationResults()
			if err != nil {
				return StatsErrorMsg{Error: err}
			}

			// Calculate statistics
			verifiedCount := 0
			totalScore := 0.0
			var lastVerification time.Time

			for _, result := range results {
				if status, ok := result["status"].(string); ok && status == "completed" {
					verifiedCount++
				}
				if score, ok := result["score"].(float64); ok {
					totalScore += score
				}
				if timestamp, ok := result["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
						if t.After(lastVerification) {
							lastVerification = t
						}
					}
				}
			}

			averageScore := 0.0
			if len(results) > 0 {
				averageScore = totalScore / float64(len(results))
			}

			if lastVerification.IsZero() {
				lastVerification = time.Now()
			}

			return StatsRefreshedMsg{
				Stats: DashboardStats{
					TotalModels:      len(models),
					TotalProviders:   len(providers),
					VerifiedModels:   verifiedCount,
					PendingModels:    len(models) - verifiedCount,
					LastVerification: lastVerification,
					AverageScore:     averageScore,
					LastRefresh:      time.Now(),
				},
			}
		},
	)
}

type StatsRefreshedMsg struct {
	Stats DashboardStats
}

type StatsErrorMsg struct {
	Error error
}
