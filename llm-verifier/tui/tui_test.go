package tui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"llm-verifier/client"
)

func TestNewApp(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	if app == nil {
		t.Error("NewApp returned nil")
	}

	if app.client != client {
		t.Error("App client not set correctly")
	}

	if len(app.screens) != 4 {
		t.Errorf("Expected 4 screens, got %d", len(app.screens))
	}

	if app.current != 0 {
		t.Errorf("Expected current screen 0, got %d", app.current)
	}
}

func TestAppInit(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	cmd := app.Init()
	if cmd == nil {
		t.Error("Init returned nil command")
	}
}

func TestAppUpdate(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	tests := []struct {
		name       string
		msg        tea.Msg
		expectsCmd bool
	}{
		{"Quit message", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, true},
		{"Screen 1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, true},
		{"Screen 2", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}, true},
		{"Screen 3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}, true},
		{"Screen 4", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}, true},
		{"Left navigation", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}, true},
		{"Right navigation", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}, true},
		{"Tab navigation", tea.KeyMsg{Type: tea.KeyTab, Runes: []rune{}}, true},
		{"Home key", tea.KeyMsg{Type: tea.KeyHome, Runes: []rune{}}, true},
		{"End key", tea.KeyMsg{Type: tea.KeyEnd, Runes: []rune{}}, true},
		{"Help key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}, false},
		{"Refresh key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := app.Update(tt.msg)
			if tt.expectsCmd && cmd == nil {
				t.Error("Expected non-nil command")
			} else if !tt.expectsCmd && cmd != nil {
				t.Error("Expected nil command")
			}
		})
	}
}

func TestAppView(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	// Test without window size
	view := app.View()
	if view == "" {
		t.Error("View returned empty string")
	}

	// Test with window size
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	view = app.View()
	if view == "" {
		t.Error("View returned empty string after window size")
	}
}

func TestAppRenderHeader(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	app.width = 100
	header := app.renderHeader()
	if header == "" {
		t.Error("renderHeader returned empty string")
	}

	// Test with different screen selections
	for i := 0; i < len(app.screens); i++ {
		app.current = i
		header = app.renderHeader()
		if header == "" {
			t.Errorf("renderHeader returned empty string for screen %d", i)
		}
	}
}

func TestAppRenderFooter(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	app.width = 100
	footer := app.renderFooter()
	if footer == "" {
		t.Error("renderFooter returned empty string")
	}
}

func TestScreenNavigation(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	// Test navigation to each screen
	for i := 0; i < len(app.screens); i++ {
		app.current = i
		view := app.View()
		if view == "" {
			t.Errorf("View returned empty string for screen %d", i)
		}
	}
}

func TestWindowResize(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},  // Terminal minimum
		{120, 40}, // Standard terminal
		{200, 60}, // Large terminal
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			app.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			view := app.View()
			if view == "" {
				t.Error("View returned empty string")
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	// Test handling of unknown message types
	_, cmd := app.Update("unknown message type")
	if cmd != nil {
		t.Error("Expected nil command for unknown message type")
	}
}

func BenchmarkAppUpdate(b *testing.B) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	}
}

func BenchmarkAppView(b *testing.B) {
	client := client.New("http://localhost:8080")
	app := NewApp(client)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.View()
	}
}
