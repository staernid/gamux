package tui

import (
	"context"
	"fmt"
	"gbe_fork_helper/gbe"
	"io"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	statePlatform state = iota
	stateAppID
	stateApplying
	stateDone
)

type model struct {
	state     state
	platforms []string
	selected  int
	textInput textinput.Model
	appID     string
	platform  string
	err       error
	applying  bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter Steam AppID"
	ti.Focus()
	ti.CharLimit = 15
	ti.Width = 20

	return model{
		state:     statePlatform,
		platforms: []string{"linux", "win64", "win32"},
		selected:  0,
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type applyMsg struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.state {
		case statePlatform:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.platforms)-1 {
					m.selected++
				}
			case "enter":
				m.platform = m.platforms[m.selected]
				m.state = stateAppID
				return m, textinput.Blink
			}

		case stateAppID:
			if msg.String() == "enter" {
				m.appID = m.textInput.Value()
				if m.appID != "" {
					m.state = stateApplying
					m.applying = true
					return m, func() tea.Msg {
						err := gbe.ApplyGBE(context.Background(), m.platform, m.appID, false)
						return applyMsg{err: err}
					}
				}
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case stateDone:
			if msg.String() == "enter" {
				return initialModel(), nil
			}
		}

	case applyMsg:
		m.state = stateDone
		m.applying = false
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")).
			MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))
	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true).
			MarginTop(1)
)

func (m model) View() string {
	var s string

	switch m.state {
	case statePlatform:
		s += titleStyle.Render("Select Platform:") + "\n"
		for i, p := range m.platforms {
			cursor := "  "
			if i == m.selected {
				cursor = "> "
				s += selectedStyle.Render(cursor + p) + "\n"
			} else {
				s += cursor + p + "\n"
			}
		}
		s += "\n(press enter to select)"

	case stateAppID:
		s += titleStyle.Render(fmt.Sprintf("Enter AppID for %s:", m.platform)) + "\n"
		s += m.textInput.View() + "\n"
		s += "\n(press enter to apply)"

	case stateApplying:
		s += titleStyle.Render(fmt.Sprintf("Applying GBE to %s (AppID: %s)...", m.platform, m.appID)) + "\n"
		s += "Please wait."

	case stateDone:
		if m.err != nil {
			s += titleStyle.Render("Error Applying GBE:") + "\n"
			s += errorStyle.Render(m.err.Error()) + "\n"
		} else {
			s += doneStyle.Render("Successfully applied GBE!") + "\n"
		}
		s += "\n(press enter to start over, or q to quit)"
	}

	return s
}

func Run() error {
	// Redirect slog output to a file when running in TUI mode to avoid screen corruption.
	logFile, err := os.OpenFile("gbe_fork_helper.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return nil
}
