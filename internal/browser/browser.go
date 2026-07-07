package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OverStackedLab/wagon/internal/filelist"
	"github.com/OverStackedLab/wagon/internal/rclone"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane struct {
	kind     filelist.Kind
	title    string
	path     string
	items    []filelist.Item
	cursor   int
	search   string
	selected map[string]bool
	err      error
	loading  bool
}

type Options struct {
	LocalPath      string
	RightPath      string
	RightKind      filelist.Kind
	AutoPickRemote bool
}

type PaneKind = filelist.Kind

const (
	LocalPane  = filelist.Local
	RemotePane = filelist.Remote
)

type Model struct {
	ctx            context.Context
	client         rclone.Client
	panes          [2]pane
	active         int
	width          int
	height         int
	status         string
	autoPickRemote bool
	copying        bool
	transfer       transferState
	searchMode     bool
	drivePicker    bool
	driveCursor    int
	driveChoices   []driveChoice
	driveErr       error
}

type paneLoadedMsg struct {
	index int
	path  string
	items []filelist.Item
	err   error
}

type remotesLoadedMsg struct {
	remotes []string
	err     error
}

type copyStepFinishedMsg struct {
	sourceIndex int
	targetIndex int
	itemIndex   int
	err         error
}

type transferTickMsg struct{}

type transferState struct {
	sourceIndex int
	targetIndex int
	targetPath  string
	targetKind  filelist.Kind
	items       []filelist.Item
	current     int
	spinner     int
	started     time.Time
}

type driveChoice struct {
	name string
	path string
}

type itemRef struct {
	index int
	item  filelist.Item
}

var (
	spinnerFrames = []string{"|", "/", "-", "\\"}

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	inactivePaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("8")).
				Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	transferStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("240"))
)

func DetectPaneKind(value string) PaneKind {
	return filelist.DetectKind(value)
}

func Run(ctx context.Context, client rclone.Client, options Options) error {
	model := NewModel(ctx, client, options)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func NewModel(ctx context.Context, client rclone.Client, options Options) Model {
	localPath := options.LocalPath
	if strings.TrimSpace(localPath) == "" {
		wd, err := os.Getwd()
		if err == nil {
			localPath = wd
		}
	}

	rightKind := options.RightKind
	rightPath := options.RightPath
	if options.AutoPickRemote {
		rightKind = filelist.Remote
	} else if rightKind == filelist.Local && strings.TrimSpace(rightPath) == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			rightPath = home
		} else {
			rightPath = localPath
		}
	}

	model := Model{
		ctx:            ctx,
		client:         client,
		autoPickRemote: options.AutoPickRemote,
		panes: [2]pane{
			{
				kind:     filelist.Local,
				title:    titleForKind(filelist.Local),
				path:     localPath,
				selected: map[string]bool{},
				loading:  true,
			},
			{
				kind:     rightKind,
				title:    titleForKind(rightKind),
				path:     rightPath,
				selected: map[string]bool{},
				loading:  true,
			},
		},
		status: "Tab switches panes. / searches. Enter opens folders. v chooses a drive.",
	}

	return model
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadPane(0)}
	if m.autoPickRemote && strings.TrimSpace(m.panes[1].path) == "" {
		cmds = append(cmds, m.loadRemotes())
	} else {
		cmds = append(cmds, m.loadPane(1))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case remotesLoadedMsg:
		if msg.err != nil {
			m.panes[1].loading = false
			m.panes[1].err = msg.err
			return m, nil
		}
		if len(msg.remotes) == 0 {
			m.panes[1].loading = false
			m.panes[1].err = fmt.Errorf("no rclone remotes configured; run rclone config")
			return m, nil
		}
		m.panes[1].kind = filelist.Remote
		m.panes[1].title = titleForKind(filelist.Remote)
		m.panes[1].path = msg.remotes[0]
		return m, m.loadPane(1)

	case transferTickMsg:
		if !m.copying {
			return m, nil
		}
		m.transfer.spinner = (m.transfer.spinner + 1) % len(spinnerFrames)
		return m, transferTick()

	case copyStepFinishedMsg:
		if msg.err != nil {
			m.copying = false
			m.status = "Copy failed: " + msg.err.Error()
			return m, nil
		}

		total := len(m.transfer.items)
		nextIndex := msg.itemIndex + 1
		if nextIndex < total {
			m.transfer.current = nextIndex
			currentItem := m.transfer.items[nextIndex]
			m.status = fmt.Sprintf("Copying %d/%d: %s", nextIndex+1, total, currentItem.Name)
			return m, m.copyTransferItem(nextIndex)
		}

		m.copying = false
		m.transfer.current = total
		m.panes[msg.sourceIndex].selected = map[string]bool{}
		m.panes[msg.targetIndex].loading = true
		m.panes[msg.targetIndex].err = nil
		m.status = fmt.Sprintf("Copied %d item(s).", total)
		return m, m.loadPane(msg.targetIndex)

	case paneLoadedMsg:
		if msg.index < 0 || msg.index >= len(m.panes) {
			return m, nil
		}
		p := &m.panes[msg.index]
		p.loading = false
		p.err = msg.err
		if msg.err == nil {
			p.path = msg.path
			p.items = msg.items
			p.cursor = clamp(p.cursor, 0, len(p.items)-1)
			syncCursorWithFilter(p)
		}
		return m, nil

	case tea.KeyMsg:
		if m.drivePicker {
			return m.updateDrivePicker(msg)
		}
		if m.searchMode {
			return m.updateSearch(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.active = 1 - m.active
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		case "enter", "right", "l":
			return m, m.openCurrent()
		case "backspace", "left", "h":
			return m, m.openParent()
		case " ":
			m.toggleSelection()
			return m, nil
		case "a":
			m.selectAll()
			return m, nil
		case "A":
			m.clearSelection()
			return m, nil
		case "/":
			m.startSearch()
			return m, nil
		case "esc":
			m.clearSearch()
			return m, nil
		case "r":
			m.status = "Refreshing " + m.activePane().title + "."
			return m, m.loadPane(m.active)
		case "c":
			return m, m.copyCurrentSelection()
		case "v":
			m.openDrivePicker()
			return m, nil
		case "s":
			m.status = "TUI sync is next. For now use: wagon sync <source> <destination>"
			return m, nil
		case "?":
			m.status = "Keys: / search, Tab pane, Enter open, Space select, c copy, v drives, a all, A clear, r refresh, q quit."
			return m, nil
		}
	}

	return m, nil
}

func (m Model) View() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	height := m.height
	if height <= 0 {
		height = 32
	}

	paneWidth := max(32, (width-3)/2)
	listHeight := max(8, height-10)

	header := headerStyle.Render("Wagon") + " " + mutedStyle.Render("rclone file manager")
	if m.drivePicker {
		return strings.Join([]string{header, m.renderDrivePicker(width, height-4)}, "\n")
	}

	left := m.renderPane(0, paneWidth, listHeight)
	right := m.renderPane(1, paneWidth, listHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	transfer := m.renderTransfer(width)
	footer := m.renderFooter(width)
	parts := []string{header, body}
	if transfer != "" {
		parts = append(parts, transfer)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func (m Model) renderPane(index int, width int, height int) string {
	p := m.panes[index]
	style := inactivePaneStyle.Width(width).Height(height)
	if index == m.active {
		style = activePaneStyle.Width(width).Height(height)
	}

	lines := []string{
		renderPathLine(p.title, p.path, width-4),
	}

	if p.loading {
		lines = append(lines, "", mutedStyle.Render("Loading..."))
		return style.Render(strings.Join(lines, "\n"))
	}
	if p.err != nil {
		lines = append(lines, "", errorStyle.Render(wrapText(p.err.Error(), width-4)))
		return style.Render(strings.Join(lines, "\n"))
	}
	if len(p.items) == 0 {
		lines = append(lines, "", mutedStyle.Render("Empty folder"))
		return style.Render(strings.Join(lines, "\n"))
	}

	if p.search != "" || (m.searchMode && index == m.active) {
		searchLine := "Search: " + p.search
		if m.searchMode && index == m.active {
			searchLine += "_"
		}
		lines = append(lines, mutedStyle.Render(truncate(searchLine, width-4)))
	}

	itemRefs := filteredItemRefs(p)
	if len(itemRefs) == 0 {
		lines = append(lines, "", mutedStyle.Render("No matches"))
		return style.Render(strings.Join(lines, "\n"))
	}

	visibleRows := max(3, height-5)
	if p.search != "" || (m.searchMode && index == m.active) {
		visibleRows = max(3, visibleRows-1)
	}
	cursorPos := cursorRefPosition(p, itemRefs)
	start := scrollStart(cursorPos, len(itemRefs), visibleRows)
	end := min(len(itemRefs), start+visibleRows)

	lines = append(lines, mutedStyle.Render(fitColumns("  Name", "Size", "Date", width-4)))
	for row := start; row < end; row++ {
		ref := itemRefs[row]
		line := m.renderItemLine(p, ref.item, ref.index == p.cursor, width-4)
		lines = append(lines, line)
	}

	selectedCount := countSelected(p)
	summary := fmt.Sprintf("%d selected", selectedCount)
	if p.search != "" {
		summary += fmt.Sprintf(" - %d match(es)", len(itemRefs))
	}
	lines = append(lines, "", mutedStyle.Render(summary))
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderItemLine(p pane, item filelist.Item, cursor bool, width int) string {
	pointer := " "
	if cursor {
		pointer = ">"
	}

	mark := " "
	if p.selected[item.Path] {
		mark = "x"
	}
	if item.IsParent {
		mark = " "
	}

	name := item.Name
	if item.IsDir && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	line := fitColumns(pointer+mark+" "+name, filelist.FormatSize(item), filelist.FormatTime(item), width)
	if p.selected[item.Path] && !item.IsParent {
		line = selectedStyle.Render(line)
	}
	if cursor {
		line = cursorStyle.Render(line)
	}
	return line
}

func (m Model) renderFooter(width int) string {
	active := m.activePane()
	selection := fmt.Sprintf("%s: %d selected", active.title, countSelected(*active))
	help := "[/] search  [Tab] pane  [Enter] open  [Space] select  [c] copy  [v] drives  [a] all  [A] clear  [r] refresh  [?] help  [q] quit"
	lines := []string{
		mutedStyle.Render(truncate(selection, width)),
		truncate(m.status, width),
		mutedStyle.Render(truncate(help, width)),
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTransfer(width int) string {
	if !m.copying || len(m.transfer.items) == 0 {
		return ""
	}

	current := clamp(m.transfer.current, 0, len(m.transfer.items)-1)
	item := m.transfer.items[current]
	frame := spinnerFrames[m.transfer.spinner%len(spinnerFrames)]
	elapsed := time.Since(m.transfer.started).Round(time.Second)
	if elapsed < 0 {
		elapsed = 0
	}

	label := fmt.Sprintf("%s Copying item %d/%d: %s -> %s  elapsed %s",
		frame,
		current+1,
		len(m.transfer.items),
		item.Name,
		m.transfer.targetPath,
		elapsed,
	)
	return transferStyle.Width(max(0, width-2)).Render(truncate(label, max(0, width-4)))
}

func (m Model) renderDrivePicker(width int, height int) string {
	pickerWidth := max(44, min(width-4, 80))
	pickerHeight := max(8, height-2)
	style := activePaneStyle.Width(pickerWidth).Height(pickerHeight)

	lines := []string{
		"Choose location for " + m.panes[m.active].title + " pane",
		mutedStyle.Render("Enter opens location. Esc cancels."),
		"",
	}

	if m.driveErr != nil {
		lines = append(lines, errorStyle.Render(wrapText(m.driveErr.Error(), pickerWidth-4)))
		return style.Render(strings.Join(lines, "\n"))
	}

	visibleRows := max(3, pickerHeight-6)
	start := scrollStart(m.driveCursor, len(m.driveChoices), visibleRows)
	end := min(len(m.driveChoices), start+visibleRows)
	for row := start; row < end; row++ {
		choice := m.driveChoices[row]
		prefix := "  "
		if row == m.driveCursor {
			prefix = "> "
		}
		line := truncate(prefix+choice.name+"  "+mutedStyle.Render(choice.path), pickerWidth-4)
		if row == m.driveCursor {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m *Model) moveCursor(delta int) {
	p := m.activePane()
	refs := filteredItemRefs(*p)
	if len(refs) == 0 {
		p.cursor = 0
		return
	}
	pos := cursorRefPosition(*p, refs)
	pos = clamp(pos+delta, 0, len(refs)-1)
	p.cursor = refs[pos].index
}

func (m *Model) openCurrent() tea.Cmd {
	p := m.activePane()
	if p.loading || p.err != nil || len(p.items) == 0 {
		return nil
	}

	item, ok := currentVisibleItem(*p)
	if !ok {
		m.status = "No matching item to open."
		return nil
	}
	if !item.IsDir {
		m.status = "Only folders can be opened in the browser."
		return nil
	}

	p.path = item.Path
	p.cursor = 0
	p.search = ""
	m.searchMode = false
	p.selected = map[string]bool{}
	p.loading = true
	p.err = nil
	return m.loadPane(m.active)
}

func (m *Model) openParent() tea.Cmd {
	p := m.activePane()
	if p.kind == filelist.Local {
		parent := localParent(p.path)
		if parent == p.path {
			return nil
		}
		p.path = parent
	} else {
		parent := filelist.RemoteParent(p.path)
		if parent == p.path {
			return nil
		}
		p.path = parent
	}
	p.cursor = 0
	p.search = ""
	m.searchMode = false
	p.selected = map[string]bool{}
	p.loading = true
	p.err = nil
	return m.loadPane(m.active)
}

func (m *Model) toggleSelection() {
	p := m.activePane()
	if p.loading || p.err != nil || len(p.items) == 0 {
		return
	}
	item, ok := currentVisibleItem(*p)
	if !ok {
		m.status = "No matching item to select."
		return
	}
	if item.IsParent {
		return
	}
	if p.selected[item.Path] {
		delete(p.selected, item.Path)
		m.status = "Unselected " + item.Name + "."
		return
	}
	p.selected[item.Path] = true
	m.status = "Selected " + item.Name + "."
}

func (m *Model) selectAll() {
	p := m.activePane()
	for _, ref := range filteredItemRefs(*p) {
		item := ref.item
		if !item.IsParent {
			p.selected[item.Path] = true
		}
	}
	m.status = fmt.Sprintf("Selected %d items.", countSelected(*p))
}

func (m *Model) clearSelection() {
	p := m.activePane()
	p.selected = map[string]bool{}
	m.status = "Selection cleared."
}

func (m *Model) startSearch() {
	p := m.activePane()
	p.search = ""
	m.searchMode = true
	m.status = "Type to filter this pane. Enter opens a match. Esc clears search."
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.clearSearch()
		return m, nil
	case "enter":
		m.searchMode = false
		return m, m.openCurrent()
	case "up", "k":
		m.moveCursor(-1)
		return m, nil
	case "down", "j":
		m.moveCursor(1)
		return m, nil
	case "backspace", "ctrl+h":
		p := m.activePane()
		p.search = dropLastRune(p.search)
		syncCursorWithFilter(p)
		m.updateSearchStatus()
		return m, nil
	case "tab":
		m.searchMode = false
		m.active = 1 - m.active
		m.status = "Search kept on the previous pane."
		return m, nil
	}

	if len(msg.Runes) > 0 {
		p := m.activePane()
		p.search += string(msg.Runes)
		syncCursorWithFilter(p)
		m.updateSearchStatus()
		return m, nil
	}

	return m, nil
}

func (m *Model) clearSearch() {
	p := m.activePane()
	if p.search == "" {
		m.searchMode = false
		m.status = "Search closed."
		return
	}

	p.search = ""
	m.searchMode = false
	syncCursorWithFilter(p)
	m.status = "Search cleared."
}

func (m *Model) updateSearchStatus() {
	p := m.activePane()
	matches := len(filteredItemRefs(*p))
	if p.search == "" {
		m.status = "Type to filter this pane. Enter opens a match. Esc clears search."
		return
	}
	m.status = fmt.Sprintf("Search: %s - %d match(es)", p.search, matches)
}

func (m *Model) openDrivePicker() {
	choices, err := localDriveChoices()
	m.drivePicker = true
	m.searchMode = false
	m.driveCursor = 0
	m.driveChoices = choices
	m.driveErr = err
	if err != nil {
		m.status = "Unable to read local drives."
		return
	}
	m.status = "Choose a local location for the active pane."
}

func (m Model) updateDrivePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.drivePicker = false
		m.status = "Drive picker closed."
		return m, nil
	case "up", "k":
		m.driveCursor = clamp(m.driveCursor-1, 0, len(m.driveChoices)-1)
		return m, nil
	case "down", "j":
		m.driveCursor = clamp(m.driveCursor+1, 0, len(m.driveChoices)-1)
		return m, nil
	case "enter":
		if m.driveErr != nil || len(m.driveChoices) == 0 {
			return m, nil
		}

		choice := m.driveChoices[m.driveCursor]
		p := &m.panes[m.active]
		p.kind = filelist.Local
		p.title = titleForKind(filelist.Local)
		p.path = choice.path
		p.cursor = 0
		p.search = ""
		p.selected = map[string]bool{}
		p.loading = true
		p.err = nil
		m.drivePicker = false
		m.status = "Opening " + choice.path + "."
		return m, m.loadPane(m.active)
	}

	return m, nil
}

func (m *Model) copyCurrentSelection() tea.Cmd {
	if m.copying {
		m.status = "Copy already running."
		return nil
	}

	sourceIndex := m.active
	targetIndex := 1 - m.active
	source := m.panes[sourceIndex]
	target := m.panes[targetIndex]

	if source.loading || source.err != nil {
		m.status = "Source pane is not ready."
		return nil
	}
	if target.loading || target.err != nil {
		m.status = "Destination pane is not ready."
		return nil
	}
	if sameLocation(source, target) {
		m.status = "Choose a different destination folder before copying."
		return nil
	}

	items := copyItems(source)
	if len(items) == 0 {
		m.status = "Select an item or move the cursor to something copyable."
		return nil
	}

	targetPath := target.path
	targetKind := target.kind
	m.copying = true
	m.transfer = transferState{
		sourceIndex: sourceIndex,
		targetIndex: targetIndex,
		targetPath:  targetPath,
		targetKind:  targetKind,
		items:       items,
		current:     0,
		started:     time.Now(),
	}
	m.status = fmt.Sprintf("Copying 1/%d: %s", len(items), items[0].Name)

	return tea.Batch(m.copyTransferItem(0), transferTick())
}

func (m Model) copyTransferItem(itemIndex int) tea.Cmd {
	client := m.client
	ctx := m.ctx
	transfer := m.transfer

	return func() tea.Msg {
		if itemIndex < 0 || itemIndex >= len(transfer.items) {
			return copyStepFinishedMsg{
				sourceIndex: transfer.sourceIndex,
				targetIndex: transfer.targetIndex,
				itemIndex:   itemIndex,
				err:         fmt.Errorf("copy item index %d out of range", itemIndex),
			}
		}

		item := transfer.items[itemIndex]
		destination := filelist.Join(transfer.targetKind, transfer.targetPath, item.Name)
		_, err := client.CopyItem(ctx, item.Path, destination, item.IsDir)
		return copyStepFinishedMsg{
			sourceIndex: transfer.sourceIndex,
			targetIndex: transfer.targetIndex,
			itemIndex:   itemIndex,
			err:         err,
		}
	}
}

func transferTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return transferTickMsg{}
	})
}

func (m Model) loadPane(index int) tea.Cmd {
	p := m.panes[index]
	client := m.client
	ctx := m.ctx

	return func() tea.Msg {
		loadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if p.kind == filelist.Local {
			resolved, items, err := filelist.ListLocal(p.path)
			return paneLoadedMsg{index: index, path: resolved, items: items, err: err}
		}

		if strings.TrimSpace(p.path) == "" {
			return paneLoadedMsg{index: index, path: p.path, err: fmt.Errorf("no remote selected")}
		}

		entries, err := client.LSJSON(loadCtx, p.path)
		if err != nil {
			return paneLoadedMsg{index: index, path: p.path, err: err}
		}
		return paneLoadedMsg{index: index, path: p.path, items: filelist.FromRemoteEntries(p.path, entries)}
	}
}

func (m Model) loadRemotes() tea.Cmd {
	client := m.client
	ctx := m.ctx
	return func() tea.Msg {
		loadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		remotes, err := client.ListRemotes(loadCtx)
		return remotesLoadedMsg{remotes: remotes, err: err}
	}
}

func (m *Model) activePane() *pane {
	return &m.panes[m.active]
}

func renderPathLine(title string, path string, width int) string {
	if strings.TrimSpace(path) == "" {
		path = "-"
	}
	return truncate(title+": "+path, width)
}

func fitColumns(name string, size string, date string, width int) string {
	sizeWidth := 9
	dateWidth := 6
	spacing := 2
	nameWidth := max(8, width-sizeWidth-dateWidth-spacing*2)
	return fmt.Sprintf("%-*s  %*s  %*s",
		nameWidth,
		truncate(name, nameWidth),
		sizeWidth,
		truncate(size, sizeWidth),
		dateWidth,
		truncate(date, dateWidth),
	)
}

func wrapText(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	var lines []string
	remaining := value
	for len(remaining) > width {
		lines = append(lines, remaining[:width])
		remaining = remaining[width:]
	}
	if remaining != "" {
		lines = append(lines, remaining)
	}
	return strings.Join(lines, "\n")
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "."
}

func scrollStart(cursor int, total int, visible int) int {
	if total <= visible {
		return 0
	}
	half := visible / 2
	start := cursor - half
	if start < 0 {
		return 0
	}
	maxStart := total - visible
	if start > maxStart {
		return maxStart
	}
	return start
}

func countSelected(p pane) int {
	return len(p.selected)
}

func filteredItemRefs(p pane) []itemRef {
	query := strings.ToLower(strings.TrimSpace(p.search))
	refs := make([]itemRef, 0, len(p.items))
	for index, item := range p.items {
		if query != "" {
			if item.IsParent {
				continue
			}
			if !strings.Contains(strings.ToLower(item.Name), query) {
				continue
			}
		}
		refs = append(refs, itemRef{index: index, item: item})
	}
	return refs
}

func cursorRefPosition(p pane, refs []itemRef) int {
	for index, ref := range refs {
		if ref.index == p.cursor {
			return index
		}
	}
	return 0
}

func syncCursorWithFilter(p *pane) {
	refs := filteredItemRefs(*p)
	if len(refs) == 0 {
		p.cursor = 0
		return
	}
	for _, ref := range refs {
		if ref.index == p.cursor {
			return
		}
	}
	p.cursor = refs[0].index
}

func currentVisibleItem(p pane) (filelist.Item, bool) {
	refs := filteredItemRefs(p)
	if len(refs) == 0 {
		return filelist.Item{}, false
	}
	for _, ref := range refs {
		if ref.index == p.cursor {
			return ref.item, true
		}
	}
	return refs[0].item, true
}

func copyItems(p pane) []filelist.Item {
	if len(p.items) == 0 {
		return nil
	}

	if len(p.selected) == 0 {
		item, ok := currentVisibleItem(p)
		if !ok {
			return nil
		}
		if item.IsParent {
			return nil
		}
		return []filelist.Item{item}
	}

	items := make([]filelist.Item, 0, len(p.selected))
	for _, item := range p.items {
		if p.selected[item.Path] && !item.IsParent {
			items = append(items, item)
		}
	}
	return items
}

func dropLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func sameLocation(left pane, right pane) bool {
	return left.kind == right.kind && strings.TrimRight(left.path, "/") == strings.TrimRight(right.path, "/")
}

func titleForKind(kind filelist.Kind) string {
	if kind == filelist.Remote {
		return "Remote"
	}
	return "Local"
}

func localDriveChoices() ([]driveChoice, error) {
	choices := []driveChoice{
		{name: "Computer root", path: string(os.PathSeparator)},
	}

	if home, err := os.UserHomeDir(); err == nil {
		choices = append(choices, driveChoice{name: "Home", path: home})
	}

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return choices, err
	}

	var volumes []driveChoice
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		volumes = append(volumes, driveChoice{
			name: entry.Name(),
			path: filepath.Join("/Volumes", entry.Name()),
		})
	}
	sort.SliceStable(volumes, func(i, j int) bool {
		return strings.ToLower(volumes[i].name) < strings.ToLower(volumes[j].name)
	})

	choices = append(choices, volumes...)
	return choices, nil
}

func localParent(value string) string {
	resolved, err := filelist.ResolveLocal(value)
	if err != nil {
		return value
	}
	return filepath.Dir(resolved)
}

func clamp(value int, low int, high int) int {
	if high < low {
		return 0
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
