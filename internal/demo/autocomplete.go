package demo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Autocomplete runs the Bubble Tea autocomplete example as a codectl demo.
func Autocomplete() error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}

type gotReposSuccessMsg []repo
// gotReposErrMsg removed (unused)

type repo struct {
	Name string `json:"name"`
}

// getRepos returns mocked data to avoid network requests during the demo.
func getRepos() tea.Msg {
	repos := []repo{
		{Name: "bubbletea"},
		{Name: "bubbles"},
		{Name: "lipgloss"},
		{Name: "glamour"},
		{Name: "wish"},
		{Name: "huh"},
		{Name: "soft-serve"},
		{Name: "vhs"},
		{Name: "charm"},
		{Name: "gum"},
	}
	return gotReposSuccessMsg(repos)
}

type model struct {
	textInput textinput.Model
	help      help.Model
	keymap    keymap
}

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}
func (k keymap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "repository"
	ti.Prompt = "charmbracelet/"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 20
	ti.ShowSuggestions = true

	h := help.New()

	km := keymap{}

	return model{textInput: ti, help: h, keymap: km}
}

func (m model) Init() tea.Cmd { return tea.Batch(getRepos, textinput.Blink) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case gotReposSuccessMsg:
		var suggestions []string
		for _, r := range msg {
			suggestions = append(suggestions, r.Name)
		}
		m.textInput.SetSuggestions(suggestions)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"Pick a Charm™ repo:\n\n  %s\n\n%s\n\n",
		m.textInput.View(),
		m.help.View(m.keymap),
	)
}
