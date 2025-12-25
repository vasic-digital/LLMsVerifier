package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"llm-verifier/client"
	"llm-verifier/tui/screens"
)

type App struct {
	client  *client.Client
	screens []tea.Model
	current int
	width   int
	height  int
}

func NewApp(client *client.Client) *App {
	return &App{
		client: client,
		screens: []tea.Model{
			screens.NewDashboardScreen(client),
			screens.NewModelsScreen(client),
			screens.NewProvidersScreen(client),
			screens.NewVerificationScreen(client),
		},
		current: 0,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.screens[a.current].Init(),
		tea.EnterAltScreen,
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "1", "F1":
			a.current = 0
			return a, a.screens[a.current].Init()
		case "2", "F2":
			a.current = 1
			return a, a.screens[a.current].Init()
		case "3", "F3":
			a.current = 2
			return a, a.screens[a.current].Init()
		case "4", "F4":
			a.current = 3
			return a, a.screens[a.current].Init()
		case "left", "h":
			if a.current > 0 {
				a.current--
				return a, a.screens[a.current].Init()
			}
		case "right", "l":
			if a.current < len(a.screens)-1 {
				a.current++
				return a, a.screens[a.current].Init()
			}
		case "tab":
			if a.current < len(a.screens)-1 {
				a.current++
			} else {
				a.current = 0
			}
			return a, a.screens[a.current].Init()
		case "home":
			a.current = 0
			return a, a.screens[a.current].Init()
		case "end":
			a.current = len(a.screens) - 1
			return a, a.screens[a.current].Init()
		case "?":
			return a, a.showHelp()
		case "r", "R":
			return a, a.refreshAllScreens()
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	}

	var cmd tea.Cmd
	a.screens[a.current], cmd = a.screens[a.current].Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	header := a.renderHeader()
	footer := a.renderFooter()
	content := a.screens[a.current].View()

	contentHeight := a.height - lipgloss.Height(header) - lipgloss.Height(footer)
	contentStyle := lipgloss.NewStyle().Height(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		contentStyle.Render(content),
		footer,
	)
}

func (a *App) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Render("LLM Verifier TUI")

	navItems := []string{
		"1. Dashboard",
		"2. Models",
		"3. Providers",
		"4. Verification",
	}

	nav := ""
	for i, item := range navItems {
		if i == a.current {
			nav += lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				Render(item)
		} else {
			nav += lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(item)
		}
		if i < len(navItems)-1 {
			nav += " | "
		}
	}

	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 2)

	return headerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			title,
			lipgloss.NewStyle().Width(a.width-lipgloss.Width(title)-2).Align(lipgloss.Right).Render(nav),
		),
	)
}

func (a *App) renderFooter() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("←/→/Tab: Navigate | 1-4/F1-F4: Jump to screen | ?: Help | r: Refresh | q: Quit")

	footerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 2)

	return footerStyle.Render(help)
}

func (a *App) showHelp() tea.Cmd {
	return func() tea.Msg {
		// This would show a help modal or screen
		// For now, we'll just refresh the current screen
		return nil
	}
}

func (a *App) refreshAllScreens() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.screens {
		cmds = append(cmds, a.screens[i].Init())
	}
	return tea.Batch(cmds...)
}
