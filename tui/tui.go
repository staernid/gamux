package tui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/steamshortcut"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMode state = iota
	statePlatform
	stateExePath
	stateAppID
	stateApplying
	stateDone
)

type mode int

const (
	modeGBE mode = iota
	modeLutris
	modeSteam
)

type model struct {
	state     state
	mode      mode
	modes     []string
	platforms []string
	selected  int
	textInput textinput.Model
	appID     string
	platform  string
	exePath   string
	portable  bool
	err       error
	applying  bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	return model{
		state:     stateMode,
		modes:     []string{"Apply GBE (DRM Removal)", "Add Game to Lutris", "Add Non-Steam Game Shortcut"},
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
		case stateMode:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.modes)-1 {
					m.selected++
				}
			case "enter":
				m.mode = mode(m.selected)
				m.selected = 0
				if m.mode == modeGBE {
					m.state = statePlatform
				} else {
					m.state = stateExePath
					m.textInput.Placeholder = "Enter executable absolute path"
					m.textInput.SetValue("")
					return m, textinput.Blink
				}
			}

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
				m.textInput.Placeholder = "Enter Steam AppID"
				m.textInput.SetValue("")
				return m, textinput.Blink
			}

		case stateExePath:
			if msg.String() == "enter" {
				m.exePath = strings.TrimSpace(m.textInput.Value())
				if m.exePath != "" {
					m.state = stateAppID
					m.textInput.Placeholder = "Enter Steam AppID (optional)"
					m.textInput.SetValue("")
					return m, textinput.Blink
				}
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case stateAppID:
			if msg.String() == "enter" {
				m.appID = strings.TrimSpace(m.textInput.Value())
				if m.mode == modeGBE && m.appID == "" {
					return m, nil
				}
				m.state = stateApplying
				m.applying = true

				return m, func() tea.Msg {
					ctx := context.Background()
					var err error
					switch m.mode {
					case modeGBE:
						err = gbe.ApplyGBE(ctx, m.platform, m.appID, false, m.portable)
					case modeLutris:
						name := ""
						if m.appID != "" {
							if n, e := steam.FetchAppName(ctx, m.appID); e == nil {
								name = n
							}
						}
						if name == "" {
							name = strings.TrimSuffix(filepath.Base(m.exePath), filepath.Ext(m.exePath))
						}
						home, e := os.UserHomeDir()
						if e != nil {
							err = e
							break
						}
						lcfg := lutris.Config{
							Name:     name,
							GamePath: m.exePath,
							Runner:   "linux",
						}
						err = lutris.Write(lcfg, filepath.Join(home, config.LutrisDir))
					case modeSteam:
						name := ""
						if m.appID != "" {
							if n, e := steam.FetchAppName(ctx, m.appID); e == nil {
								name = n
							}
						}
						if name == "" {
							name = strings.TrimSuffix(filepath.Base(m.exePath), filepath.Ext(m.exePath))
						}
						scfg := steamshortcut.ShortcutConfig{
							Name:    name,
							ExePath: m.exePath,
							AppID:   m.appID,
						}
						err = steamshortcut.RegisterShortcut(ctx, scfg, false)
					}
					return applyMsg{err: err}
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
	case stateMode:
		s += titleStyle.Render("Select Workflow Mode:") + "\n"
		for i, modeName := range m.modes {
			cursor := "  "
			if i == m.selected {
				cursor = "> "
				s += selectedStyle.Render(cursor+modeName) + "\n"
			} else {
				s += cursor + modeName + "\n"
			}
		}
		s += "\n(press enter to select)"

	case statePlatform:
		s += titleStyle.Render("Select Platform:") + "\n"
		for i, p := range m.platforms {
			cursor := "  "
			if i == m.selected {
				cursor = "> "
				s += selectedStyle.Render(cursor+p) + "\n"
			} else {
				s += cursor + p + "\n"
			}
		}
		s += "\n(press enter to select)"

	case stateExePath:
		s += titleStyle.Render("Executable Path:") + "\n"
		s += m.textInput.View() + "\n"
		s += "\n(press enter to confirm)"

	case stateAppID:
		if m.mode == modeGBE {
			s += titleStyle.Render(fmt.Sprintf("Enter AppID for %s:", m.platform)) + "\n"
		} else {
			s += titleStyle.Render("Enter Steam AppID (optional, press Enter to skip):") + "\n"
		}
		s += m.textInput.View() + "\n"
		s += "\n(press enter to apply)"

	case stateApplying:
		s += titleStyle.Render("Processing request...") + "\n"
		s += "Please wait."

	case stateDone:
		if m.err != nil {
			s += titleStyle.Render("Error:") + "\n"
			s += errorStyle.Render(m.err.Error()) + "\n"
		} else {
			s += doneStyle.Render("Operation completed successfully!") + "\n"
		}
		s += "\n(press enter to start over, or q to quit)"
	}

	return s
}

func Run() error {
	logFile, err := os.OpenFile("gamux.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
