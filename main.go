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
	height       int
	filter       string
	filtered     []int
	targetOffset int
	actionOffset int
}

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Bold(true)
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	checkedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)
	itemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	noteStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	filterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	sectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)
	headerBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Background(lipgloss.Color("#44475A"))
	headerTitle  = headerBar.Copy().Bold(true)
	headerMeta   = headerBar.Copy().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	activeStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#3B3F52"))
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
		m.height = msg.Height
		switch m.step {
		case stepTargets:
			m.ensureTargetVisible()
		case stepAction:
			m.ensureActionVisible()
		}
	}
	return m, nil
}

func (m model) updateTargets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveTargetCursor(-1)
	case "down", "j":
		m.moveTargetCursor(1)
	case " ":
		m.toggleSelection(m.cursor)
		if m.hasSelection() && m.note != "" {
			m.note = ""
		}
	case "enter":
		if !m.hasSelection() {
			m.note = "Select at least one target (or 'all') with Space."
			m.ensureTargetVisible()
			return m, nil
		}
		m.note = ""
		m.step = stepAction
		m.cursor = 0
		m.actionCursor = 0
		m.actionOffset = 0
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
	m.ensureTargetVisible()
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
	m.ensureActionVisible()
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
	height := m.currentHeight()
	writeHeader(&b, inner, "TARGET SELECTOR")
	writeWrapped(&b, filterStyle, fmt.Sprintf("FILTER: %s", m.filter), inner)
	b.WriteString("\n")
	indexes := m.filtered
	if len(indexes) == 0 {
		indexes = make([]int, len(m.targets))
		for i := range m.targets {
			indexes[i] = i
		}
	}
	var lines []string
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
				if m.cursor == i {
					out = activeStyle.Render(out)
				}
				lines = append(lines, out)
				continue
			}
			indent := strings.Repeat(" ", prefixLen)
			out := indent + itemStyle.Render(line)
			if m.cursor == i {
				out = activeStyle.Render(out)
			}
			lines = append(lines, out)
		}
	}
	if height > 0 {
		visible, showNote := m.targetVisibleRows(inner, height)
		start, end := clampSlice(m.targetOffset, visible, len(lines))
		for _, line := range lines[start:end] {
			b.WriteString(line + "\n")
		}
		if showNote {
			b.WriteString("\n")
			writeWrapped(&b, noteStyle, m.note, inner)
		}
		return padToHeight(b.String(), height)
	}
	for _, line := range lines {
		b.WriteString(line + "\n")
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
	height := m.currentHeight()
	writeHeader(&b, inner, "ACTION SELECTOR")
	b.WriteString("\n")
	var lines []string
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
				if m.actionCursor == i {
					out = activeStyle.Render(out)
				}
				lines = append(lines, out)
				continue
			}
			indent := strings.Repeat(" ", prefixLen)
			out := indent + itemStyle.Render(line)
			if m.actionCursor == i {
				out = activeStyle.Render(out)
			}
			lines = append(lines, out)
		}
	}
	if height > 0 {
		visible := m.actionVisibleRows(height)
		start, end := clampSlice(m.actionOffset, visible, len(lines))
		for _, line := range lines[start:end] {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
		return b.String()
	}
	for _, line := range lines {
		b.WriteString(line + "\n")
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

func (m model) currentHeight() int {
	if m.height > 0 {
		return m.height
	}
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return height
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

func (m model) targetIndexes() []int {
	if len(m.filtered) == 0 {
		indexes := make([]int, len(m.targets))
		for i := range m.targets {
			indexes[i] = i
		}
		return indexes
	}
	return m.filtered
}

func (m *model) moveTargetCursor(delta int) {
	indexes := m.targetIndexes()
	if len(indexes) == 0 {
		return
	}
	pos := 0
	found := false
	for i, idx := range indexes {
		if idx == m.cursor {
			pos = i
			found = true
			break
		}
	}
	if !found {
		pos = 0
	}
	next := pos + delta
	if next < 0 {
		next = 0
	}
	if next >= len(indexes) {
		next = len(indexes) - 1
	}
	m.cursor = indexes[next]
}

func (m model) targetLabelWidth(inner int) int {
	prefixPlain := fmt.Sprintf("%s %s ", ">", "[ ]")
	prefixLen := len([]rune(prefixPlain))
	labelWidth := inner - prefixLen
	if labelWidth < 1 {
		labelWidth = 1
	}
	return labelWidth
}

func (m model) targetCursorLine(inner int, indexes []int) (int, int) {
	line := 0
	labelWidth := m.targetLabelWidth(inner)
	for _, i := range indexes {
		item := m.targets[i]
		lines := wrapLines(item.Label, labelWidth)
		if i == m.cursor {
			if len(lines) == 0 {
				return line, 1
			}
			return line, len(lines)
		}
		line += len(lines)
	}
	return 0, 1
}

func (m model) targetTotalLines(inner int, indexes []int) int {
	total := 0
	labelWidth := m.targetLabelWidth(inner)
	for _, i := range indexes {
		total += len(wrapLines(m.targets[i].Label, labelWidth))
	}
	return total
}

func (m model) targetVisibleRows(inner int, height int) (int, bool) {
	if height <= 0 {
		return 0, false
	}
	headerLines := 2
	filterLines := len(wrapLines(fmt.Sprintf("FILTER: %s", m.filter), inner))
	blank := 1
	noteLines := 0
	showNote := m.note != ""
	if showNote {
		noteLines = 1 + len(wrapLines(m.note, inner))
	}
	available := height - headerLines - filterLines - blank - noteLines
	if available < 1 && showNote {
		showNote = false
		noteLines = 0
		available = height - headerLines - filterLines - blank
	}
	if available < 1 {
		available = 1
	}
	return available, showNote
}

func (m *model) ensureTargetVisible() {
	inner := m.innerWidth()
	height := m.currentHeight()
	if height <= 0 {
		return
	}
	indexes := m.targetIndexes()
	total := m.targetTotalLines(inner, indexes)
	visible, _ := m.targetVisibleRows(inner, height)
	if total <= 0 || visible <= 0 {
		m.targetOffset = 0
		return
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.targetOffset > maxOffset {
		m.targetOffset = maxOffset
	}
	cursorLine, _ := m.targetCursorLine(inner, indexes)
	if cursorLine < m.targetOffset {
		m.targetOffset = cursorLine
		return
	}
	if cursorLine >= m.targetOffset+visible {
		m.targetOffset = cursorLine - visible + 1
		if m.targetOffset < 0 {
			m.targetOffset = 0
		}
	}
}

func (m model) actionLabelWidth(inner int) int {
	prefixPlain := fmt.Sprintf("%s ", ">")
	prefixLen := len([]rune(prefixPlain))
	labelWidth := inner - prefixLen
	if labelWidth < 1 {
		labelWidth = 1
	}
	return labelWidth
}

func (m model) actionCursorLine(inner int) int {
	line := 0
	labelWidth := m.actionLabelWidth(inner)
	for i, item := range actions {
		lines := wrapLines(item, labelWidth)
		if i == m.actionCursor {
			if len(lines) == 0 {
				return line
			}
			return line
		}
		line += len(lines)
	}
	return 0
}

func (m model) actionTotalLines(inner int) int {
	total := 0
	labelWidth := m.actionLabelWidth(inner)
	for _, item := range actions {
		total += len(wrapLines(item, labelWidth))
	}
	return total
}

func (m model) actionVisibleRows(height int) int {
	if height <= 0 {
		return 0
	}
	headerLines := 2
	available := height - headerLines
	if available < 1 {
		available = 1
	}
	return available
}

func (m *model) ensureActionVisible() {
	inner := m.innerWidth()
	height := m.currentHeight()
	if height <= 0 {
		return
	}
	total := m.actionTotalLines(inner)
	visible := m.actionVisibleRows(height)
	if total <= 0 || visible <= 0 {
		m.actionOffset = 0
		return
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.actionOffset > maxOffset {
		m.actionOffset = maxOffset
	}
	cursorLine := m.actionCursorLine(inner)
	if cursorLine < m.actionOffset {
		m.actionOffset = cursorLine
		return
	}
	if cursorLine >= m.actionOffset+visible {
		m.actionOffset = cursorLine - visible + 1
		if m.actionOffset < 0 {
			m.actionOffset = 0
		}
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

func clampSlice(offset int, visible int, total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if offset < 0 {
		offset = 0
	}
	if visible <= 0 || offset >= total {
		return 0, total
	}
	end := offset + visible
	if end > total {
		end = total
	}
	return offset, end
}

func padToHeight(out string, height int) string {
	if height <= 0 {
		return out
	}
	lines := strings.Count(out, "\n")
	if lines >= height {
		return out
	}
	return out + strings.Repeat("\n", height-lines)
}

func makeLine(width int, ch byte) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(string(ch), width)
}

func writeHeader(b *strings.Builder, width int, subtitle string) {
	if width <= 0 {
		writeWrapped(b, headerTitle, " TFZ ", width)
		writeWrapped(b, sectionStyle, subtitle, width)
		return
	}
	left := " TFZ "
	right := " " + subtitle + " "
	leftLen := len([]rune(left))
	rightLen := len([]rune(right))
	fill := width - leftLen - rightLen
	if fill < 0 {
		b.WriteString(headerTitle.Render(left))
		b.WriteString(headerMeta.Render(right) + "\n")
	} else {
		b.WriteString(headerTitle.Render(left))
		b.WriteString(headerBar.Render(strings.Repeat(" ", fill)))
		b.WriteString(headerMeta.Render(right) + "\n")
	}
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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Print(helpString())
			return
		case "-v", "--version", "version":
			fmt.Println(versionString())
			return
		}
	}

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

func versionString() string {
	v := version
	if v != "" && v != "dev" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	out := "tfz " + v
	if commit != "" && commit != "none" {
		out += " (" + commit + ")"
	}
	if date != "" && date != "unknown" {
		out += " " + date
	}
	return out
}

func helpString() string {
	return `tfz - A small, fast TUI for running Terraform plan/apply with optional targets.

Usage:
  tfz

Options:
  -h, --help     Show this help
  -v, --version  Print version
`
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
