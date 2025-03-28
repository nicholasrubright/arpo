package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"math/rand"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/charmbracelet/bubbles/spinner"
)

const (
	columnKeyID	= "id"
	columnKeyName = "name"
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle      = helpStyle.UnsetMargins()
	durationStyle = dotStyle
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	checkMark	= lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

const dev_path string = ""


type projectsMsg struct{ projects []project }
type errMsg struct{ err error }
func (e errMsg) Error() string { return e.err.Error() }

type archivedProjectMsg struct {
	duration time.Duration
	project project
}

func (ar archivedProjectMsg) String() string {
	if ar.duration == 0 {
		return dotStyle.Render(strings.Repeat(".", 30))
	}
	return fmt.Sprintf("📦 Moved %s %s", ar.project.name, durationStyle.Render(ar.duration.String()))
}



type model struct {
	table table.Model
	spinner spinner.Model

	projectMap map[string]project
	projects []project
	completed []archivedProjectMsg
	index	int
	arpo_path	string
	
	archiving bool

	done	bool
	quitting bool
	err error
}

func initialModel() model {
	
	const numLastResults = 5
	s := spinner.New()
	s.Style = spinnerStyle

	columns := []table.Column{
		table.NewColumn(columnKeyName, "Name", 20),
	}

	keys := table.DefaultKeyMap()
	keys.RowDown.SetKeys("j", "down", "s")
	keys.RowUp.SetKeys("k", "up", "w")
	keys.RowSelectToggle.SetKeys(" ")

	t := table.New(columns).
		HeaderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)).
		Focused(true).
		SelectableRows(true).
		WithSelectedText(" ", "✓").
		WithKeyMap(keys).
		WithPageSize(10).
		
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(lipgloss.Color("#a38")).
				Foreground(lipgloss.Color("#a7a")).
				Align(lipgloss.Left),
		)

	return model{
		table: t,
		spinner: s,
		archiving: false,
		projectMap: make(map[string]project),
		completed: make([]archivedProjectMsg, 5),
		arpo_path: filepath.Join(dev_path, "arpo"),
	}

}

func (m model) Init() (tea.Cmd) {
	return tea.Batch(getProjects())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	
	if !m.archiving {
		return updateChoices(msg, m)
	}

	return updateArchive(msg, m)
}

func getProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := GetProjectDirectories()

		if err != nil {
			panic(err)
		}
	
		return projectsMsg{projects}
	}
}

func updateChoices(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	
	var (
		cmd tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case projectsMsg:
		var rows []table.Row
		for i, project := range msg.projects {
			row := table.NewRow(table.RowData{
				columnKeyID: i,
				columnKeyName: project.name,
			})
			m.projectMap[project.name] = project
			rows = append(rows, row)
		}
		m.projects = msg.projects
		m.table = m.table.WithRows(rows)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.archiving = true	
			for _, row := range m.table.SelectedRows() {
				selectedProject := row.Data[columnKeyName].(string)
				cmds = append(cmds, archiveProject(m.projectMap[selectedProject], m.arpo_path))
			}
			cmds = append(cmds, m.spinner.Tick)

		}
	
	}

	return m, tea.Batch(cmds...)

}

func archiveProject(proj project, arpo_path string) tea.Cmd {
	return func() tea.Msg {
		pause := time.Duration(rand.Int63n(899)+100) * time.Millisecond // nolint:gosec
		time.Sleep(pause)

		new_path := filepath.Join(arpo_path, proj.name)

		err := MoveDirectories(proj.path, new_path)

		if err != nil {
			panic(err)
		}

		return archivedProjectMsg{project: proj, duration: pause }
	}
}

func updateArchive(msg tea.Msg, m model) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case archivedProjectMsg:
		if m.index >= len(m.projects)-1 {
			m.done = true
			return m, tea.Quit
		}
		m.index++
		m.completed = append(m.completed[1:], msg)
		return m, tea.Batch(
			archiveProject(m.projects[m.index], m.arpo_path),
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, tea.Batch(cmd)
	}

	return m, nil
}

func (m model) View() string {

	var s string

	if !m.archiving {
		s = choicesView(m)
	} else {
		s = archivingView(m)
	}

	return s
}

// Select projects to move
func choicesView(m model) string {
	var s string

	s += m.table.View() + "\n"

	if !m.quitting {
		s += helpStyle.Render("Press q to exit")
	}

	return appStyle.Render(s)
}


func archivingView(m model) string {
	var s string

	if m.quitting  {
		s += "Archiving Canceled"
	} else if m.done {
		s +=  "✓ Archiving Completed!"
	} else {
		s += m.spinner.View() + " Archiving projects..."
	}

	s += "\n\n"

	for _, res := range m.completed {
		s += res.String() + "\n"
	}

	if m.done {
		s += "\nDone!" + "\n"
	}

	if !m.quitting {
		s += helpStyle.Render("Press q to exit")
	}

	if m.quitting {
		s += "\n"
	}


	return appStyle.Render(s)
}

func main() {

	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}

