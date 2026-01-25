package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type step int

const (
	stepTargets step = iota
	stepAction
)

type targetItem struct {
	Label    string
	Selected bool
}

type model struct {
	step         step
	cursor       int
	actionCursor int
	targets      []targetItem
	action       string
	note         string
	width        int
	filter       string
	filtered     []int
}

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Bold(true)
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
	checkedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	itemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	noteStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))
	filterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case stepTargets:
			return m.updateTargets(msg)
		case stepAction:
			return m.updateAction(msg)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m model) updateTargets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.targets)-1 {
			m.cursor++
		}
	case " ":
		m.toggleSelection(m.cursor)
	case "enter":
		if !m.hasSelection() {
			m.selectAllOnly()
		}
		m.step = stepAction
		m.cursor = 0
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.rebuildFilter()
		}
	default:
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.filter += string(msg.Runes)
			m.rebuildFilter()
		}
	}
	return m, nil
}

func (m model) updateAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	case "down", "j":
		if m.actionCursor < len(actions)-1 {
			m.actionCursor++
		}
	case "enter":
		m.action = actions[m.actionCursor]
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	switch m.step {
	case stepTargets:
		return m.viewTargets()
	case stepAction:
		return m.viewAction()
	default:
		return ""
	}
}

func (m model) viewTargets() string {
	var b strings.Builder
	inner := m.innerWidth()
	writeWrapped(&b, titleStyle, "Select targets", inner)
	writeWrapped(&b, noteStyle, "Type: filter, Space: toggle, Enter: confirm, q: quit", inner)
	writeWrapped(&b, filterStyle, fmt.Sprintf("Filter: %s", m.filter), inner)
	b.WriteString("\n")
	indexes := m.filtered
	if len(indexes) == 0 {
		indexes = make([]int, len(m.targets))
		for i := range m.targets {
			indexes[i] = i
		}
	}
	for _, i := range indexes {
		item := m.targets[i]
		cursorPlain := " "
		cursorStyled := " "
		if m.cursor == i {
			cursorPlain = ">"
			cursorStyled = cursorStyle.Render(">")
		}
		checkPlain := "[ ]"
		checkStyled := itemStyle.Render("[ ]")
		if item.Selected {
			checkPlain = "[x]"
			checkStyled = checkedStyle.Render("[x]")
		}
		prefixPlain := fmt.Sprintf("%s %s ", cursorPlain, checkPlain)
		prefixLen := len([]rune(prefixPlain))
		labelWidth := inner - prefixLen
		if labelWidth < 1 {
			labelWidth = 1
		}
		labelLines := wrapLines(item.Label, labelWidth)
		for li, line := range labelLines {
			if li == 0 {
				out := fmt.Sprintf("%s %s %s", cursorStyled, checkStyled, itemStyle.Render(line))
				b.WriteString(out + "\n")
				continue
			}
			indent := strings.Repeat(" ", prefixLen)
			b.WriteString(indent + itemStyle.Render(line) + "\n")
		}
	}
	if m.note != "" {
		b.WriteString("\n")
		writeWrapped(&b, noteStyle, m.note, inner)
	}
	return b.String()
}

func (m model) viewAction() string {
	var b strings.Builder
	inner := m.innerWidth()
	writeWrapped(&b, titleStyle, "Select action", inner)
	writeWrapped(&b, noteStyle, "Enter: run, q: quit", inner)
	b.WriteString("\n")
	for i, item := range actions {
		cursorPlain := " "
		cursorStyled := " "
		if m.actionCursor == i {
			cursorPlain = ">"
			cursorStyled = cursorStyle.Render(">")
		}
		prefixPlain := fmt.Sprintf("%s ", cursorPlain)
		prefixLen := len([]rune(prefixPlain))
		labelWidth := inner - prefixLen
		if labelWidth < 1 {
			labelWidth = 1
		}
		labelLines := wrapLines(item, labelWidth)
		for li, line := range labelLines {
			if li == 0 {
				out := fmt.Sprintf("%s %s", cursorStyled, itemStyle.Render(line))
				b.WriteString(out + "\n")
				continue
			}
			indent := strings.Repeat(" ", prefixLen)
			b.WriteString(indent + itemStyle.Render(line) + "\n")
		}
	}
	return b.String()
}

func (m model) innerWidth() int {
	frameWidth := m.currentWidth()
	if frameWidth <= 0 {
		return 0
	}
	return frameWidth
}

func (m model) currentWidth() int {
	if m.width > 0 {
		return m.width
	}
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return width
}

func (m *model) rebuildFilter() {
	m.filtered = m.filtered[:0]
	if m.filter == "" {
		return
	}
	query := strings.ToLower(m.filter)
	for i, item := range m.targets {
		if fuzzyMatch(strings.ToLower(item.Label), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	if len(m.filtered) == 0 {
		m.cursor = 0
		return
	}
	valid := false
	for _, idx := range m.filtered {
		if idx == m.cursor {
			valid = true
			break
		}
	}
	if !valid {
		m.cursor = m.filtered[0]
	}
}

func fuzzyMatch(text, query string) bool {
	if query == "" {
		return true
	}
	ti := 0
	for _, r := range query {
		found := false
		for ti < len(text) {
			if rune(text[ti]) == r {
				found = true
				ti++
				break
			}
			ti++
		}
		if !found {
			return false
		}
	}
	return true
}

func writeWrapped(b *strings.Builder, style lipgloss.Style, text string, width int) {
	lines := wrapLines(text, width)
	for _, line := range lines {
		b.WriteString(style.Render(line) + "\n")
	}
}

func wrapLines(text string, width int) []string {
	if width <= 0 {
		return strings.Split(text, "\n")
	}
	var out []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > width {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, string(runes))
	}
	return out
}

func (m *model) toggleSelection(index int) {
	if index == 0 {
		if !m.targets[0].Selected {
			m.selectAllOnly()
		} else {
			m.targets[0].Selected = false
		}
		return
	}
	if m.targets[0].Selected {
		m.targets[0].Selected = false
	}
	m.targets[index].Selected = !m.targets[index].Selected
}

func (m *model) selectAllOnly() {
	for i := range m.targets {
		m.targets[i].Selected = false
	}
	if len(m.targets) > 0 {
		m.targets[0].Selected = true
	}
}

func (m model) hasSelection() bool {
	for _, item := range m.targets {
		if item.Selected {
			return true
		}
	}
	return false
}

func (m model) selectedTargets() []string {
	if len(m.targets) == 0 || m.targets[0].Selected {
		return nil
	}
	var out []string
	for _, item := range m.targets[1:] {
		if item.Selected {
			out = append(out, item.Label)
		}
	}
	return out
}

var (
	reModule  = regexp.MustCompile(`^\s*module\s+"([^"]+)"`)
	reRes     = regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"`)
	actions   = []string{"plan", "apply"}
	tfExt     = ".tf"
	allTarget = "all"
)

func main() {
	targets, err := findTargets(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	items := make([]targetItem, 0, len(targets)+1)
	items = append(items, targetItem{Label: allTarget})
	for _, t := range targets {
		items = append(items, targetItem{Label: t})
	}

	note := ""
	if len(targets) == 0 {
		note = "No .tf targets found; selecting 'all' will run without -target."
	}

	m := model{
		step:    stepTargets,
		targets: items,
		note:    note,
	}
	prog := tea.NewProgram(m)
	final, err := prog.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fm, ok := final.(model)
	if !ok {
		os.Exit(1)
	}
	if fm.action == "" {
		return
	}

	args := []string{fm.action}
	selected := fm.selectedTargets()
	for _, t := range selected {
		args = append(args, "-target="+t)
	}

	fmt.Printf("terraform %s\n", strings.Join(args, " "))
	cmd := exec.Command("terraform", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func findTargets(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*"+tfExt))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	for _, path := range matches {
		if err := collectTargets(path, seen); err != nil {
			return nil, err
		}
	}

	var out []string
	for target := range seen {
		out = append(out, target)
	}
	sort.Strings(out)
	return out, nil
}

func collectTargets(path string, seen map[string]struct{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		if match := reModule.FindStringSubmatch(line); match != nil {
			seen["module."+match[1]] = struct{}{}
			continue
		}
		if match := reRes.FindStringSubmatch(line); match != nil {
			seen["resource."+match[1]+"."+match[2]] = struct{}{}
		}
	}
	return scanner.Err()
}
