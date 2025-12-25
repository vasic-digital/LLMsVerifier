package screens

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"llm-verifier/client"
)

func TestNewDashboardScreen(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	if screen == nil {
		t.Error("NewDashboardScreen returned nil")
	}

	if screen.client != client {
		t.Error("Screen client not set correctly")
	}

	if screen.stats.TotalModels != 0 {
		t.Error("Initial stats should be zero")
	}
}

func TestDashboardScreenInit(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	cmd := screen.Init()
	if cmd == nil {
		t.Error("Init returned nil command")
	}
}

func TestDashboardScreenUpdate(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"Window size", tea.WindowSizeMsg{Width: 100, Height: 50}},
		{"Refresh key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}},
		{"Tick message", tickMsg(time.Now())},
		{"Stats refreshed", StatsRefreshedMsg{Stats: DashboardStats{TotalModels: 10}}},
		{"Stats error", StatsErrorMsg{Error: nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := screen.Update(tt.msg)
			if cmd == nil && tt.name != "Stats error" {
				t.Error("Expected non-nil command")
			}
		})
	}
}

func TestDashboardScreenView(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	// Test without window size
	view := screen.View()
	if view == "" {
		t.Error("View returned empty string")
	}

	// Test with window size
	screen.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	view = screen.View()
	if view == "" {
		t.Error("View returned empty string after window size")
	}

	// Test with stats
	screen.stats = DashboardStats{
		TotalModels:      10,
		TotalProviders:   3,
		VerifiedModels:   7,
		PendingModels:    3,
		AverageScore:     85.5,
		LastVerification: time.Now(),
	}
	view = screen.View()
	if view == "" {
		t.Error("View returned empty string with stats")
	}
}

func TestDashboardScreenRefreshStats(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	cmd := screen.refreshStats()
	if cmd == nil {
		t.Error("refreshStats returned nil command")
	}
}

func TestDashboardScreenRenderStats(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	screen.stats = DashboardStats{
		TotalModels:      15,
		TotalProviders:   4,
		VerifiedModels:   12,
		PendingModels:    3,
		AverageScore:     92.3,
		LastVerification: time.Now(),
	}

	statsView := screen.renderStats()
	if statsView == "" {
		t.Error("renderStats returned empty string")
	}
}

func TestDashboardScreenRenderProgress(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	screen.stats = DashboardStats{
		TotalModels:    10,
		VerifiedModels: 7,
	}

	progressView := screen.renderProgress()
	if progressView == "" {
		t.Error("renderProgress returned empty string")
	}

	// Test with zero models
	screen.stats.TotalModels = 0
	progressView = screen.renderProgress()
	if progressView != "" {
		t.Error("renderProgress should return empty string with zero models")
	}
}

func TestDashboardScreenRenderActions(t *testing.T) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	actionsView := screen.renderActions()
	if actionsView == "" {
		t.Error("renderActions returned empty string")
	}
}

func BenchmarkDashboardScreenUpdate(b *testing.B) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	}
}

func BenchmarkDashboardScreenView(b *testing.B) {
	client := client.New("http://localhost:8080")
	screen := NewDashboardScreen(client)
	screen.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.View()
	}
}
