package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	clientapi "github.com/shaoyanji/bountystash/internal/client/api"
	"github.com/shaoyanji/bountystash/internal/http/handlers"
	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/version"
)

type mode string

const (
	modeBrowse mode = "browse"
	modeReview mode = "review"
	modeCreate mode = "create"
)

type browseItem struct {
	title  string
	desc   string
	kind   string
	source string
	id     string
}

func (i browseItem) Title() string       { return i.title }
func (i browseItem) Description() string { return i.desc }
func (i browseItem) FilterValue() string { return i.title + " " + i.desc + " " + i.kind }

type reviewItem struct {
	title       string
	desc        string
	id          string
	isSensitive bool
}

func (i reviewItem) Title() string       { return i.title }
func (i reviewItem) Description() string { return i.desc }
func (i reviewItem) FilterValue() string { return i.title + " " + i.desc + " " + i.id }

type healthMsg struct {
	status string
	err    error
}

type examplesMsg struct {
	items []clientapi.ExampleSummary
	err   error
}

type workListMsg struct {
	items []handlers.WorkListRow
	err   error
}

type reviewMsg struct {
	queue clientapi.ReviewQueue
	err   error
}

type detailMsg struct {
	text string
	err  error
}

type draftCreateMsg struct {
	result handlers.DraftCreateResult
	err    error
}

type model struct {
	api      clientapi.Client
	baseURL  string
	noColor  bool
	mode     mode
	showHelp bool
	errMsg   string
	status   string

	width  int
	height int

	examples []clientapi.ExampleSummary
	work     []handlers.WorkListRow
	review   clientapi.ReviewQueue

	browseList list.Model
	reviewList list.Model
	detail     string

	titleInput      textinput.Model
	rewardInput     textinput.Model
	scopeInput      textarea.Model
	delivInput      textarea.Model
	acceptInput     textarea.Model
	createFocus     int
	kindIndex       int
	visibilityIndex int

	lastCreatedID string
}

var kinds = []string{"bounty", "rfq", "rfp", "private_security"}
var visibilities = []string{"draft", "private", "public", "archived"}

const compiledDefaultBaseURL = "https://garnixmachine.main.nixconfig.shaoyanji.garnix.me/"

func initialModel(api clientapi.Client, baseURL string, noColor bool) model {
	browse := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	browse.Title = "Browse"
	browse.SetShowStatusBar(false)
	browse.SetFilteringEnabled(false)
	browse.SetShowHelp(false)

	review := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	review.Title = "Review Queue"
	review.SetShowStatusBar(false)
	review.SetFilteringEnabled(false)
	review.SetShowHelp(false)

	title := textinput.New()
	title.Placeholder = "Title"
	title.Prompt = ""
	title.Focus()
	title.CharLimit = 180

	reward := textinput.New()
	reward.Placeholder = "Reward or quote model"
	reward.Prompt = ""

	scope := textarea.New()
	scope.Placeholder = "Scope lines (one per line)"
	scope.SetHeight(3)

	deliv := textarea.New()
	deliv.Placeholder = "Deliverables (one per line)"
	deliv.SetHeight(3)

	accept := textarea.New()
	accept.Placeholder = "Acceptance criteria (one per line)"
	accept.SetHeight(3)

	return model{
		api:             api,
		baseURL:         baseURL,
		noColor:         noColor,
		mode:            modeBrowse,
		status:          "loading",
		browseList:      browse,
		reviewList:      review,
		titleInput:      title,
		rewardInput:     reward,
		scopeInput:      scope,
		delivInput:      deliv,
		acceptInput:     accept,
		kindIndex:       0,
		visibilityIndex: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchHealthCmd(m.api), fetchExamplesCmd(m.api), fetchWorkCmd(m.api), fetchReviewCmd(m.api))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listHeight := max(8, msg.Height-8)
		m.browseList.SetSize(max(20, msg.Width/3), listHeight)
		m.reviewList.SetSize(max(20, msg.Width/3), listHeight)
		m.scopeInput.SetWidth(max(40, msg.Width-28))
		m.delivInput.SetWidth(max(40, msg.Width-28))
		m.acceptInput.SetWidth(max(40, msg.Width-28))
		return m, nil
	case tea.KeyMsg:
		if key := msg.String(); key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		switch msg.String() {
		case "b":
			m.mode = modeBrowse
			m.showHelp = false
			return m, nil
		case "r":
			m.mode = modeReview
			m.showHelp = false
			return m, nil
		case "c":
			m.mode = modeCreate
			m.showHelp = false
			m.setCreateFocus(m.createFocus)
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+l":
			return m, tea.Batch(fetchHealthCmd(m.api), fetchExamplesCmd(m.api), fetchWorkCmd(m.api), fetchReviewCmd(m.api))
		}
	}

	switch msg := msg.(type) {
	case healthMsg:
		if msg.err != nil {
			m.status = "down"
			m.errMsg = msg.err.Error()
		} else {
			m.status = msg.status
		}
		return m, nil
	case examplesMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.examples = msg.items
		m.refreshBrowseItems()
		return m, nil
	case workListMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.work = msg.items
		m.refreshBrowseItems()
		return m, nil
	case reviewMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.review = msg.queue
		m.refreshReviewItems()
		return m, nil
	case detailMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.detail = msg.text
		m.errMsg = ""
		return m, nil
	case draftCreateMsg:
		if msg.err != nil {
			if apiErr, ok := msg.err.(*clientapi.APIError); ok && len(apiErr.ValidationErrors) > 0 {
				m.errMsg = formatValidationErrors(apiErr.ValidationErrors)
			} else {
				m.errMsg = msg.err.Error()
			}
			return m, nil
		}
		m.lastCreatedID = msg.result.ID
		m.errMsg = ""
		m.detail = formatWorkDetail(handlers.WorkDetail{
			ID:           msg.result.ID,
			Status:       msg.result.Status,
			Version:      msg.result.Version,
			ExactHash:    msg.result.ExactHash,
			QuotientHash: msg.result.QuotientHash,
			Packet:       msg.result.Packet,
		})
		m.mode = modeBrowse
		m.clearCreateForm()
		return m, tea.Batch(fetchWorkCmd(m.api), fetchReviewCmd(m.api))
	}

	switch m.mode {
	case modeBrowse:
		updated, cmd := m.browseList.Update(msg)
		m.browseList = updated
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			item, ok := m.browseList.SelectedItem().(browseItem)
			if !ok {
				return m, nil
			}
			if item.source == "example" {
				return m, fetchExampleDetailCmd(m.api, item.id)
			}
			return m, fetchWorkDetailCmd(m.api, item.id)
		}
		return m, cmd
	case modeReview:
		updated, cmd := m.reviewList.Update(msg)
		m.reviewList = updated
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			item, ok := m.reviewList.SelectedItem().(reviewItem)
			if !ok {
				return m, nil
			}
			return m, fetchWorkDetailCmd(m.api, item.id)
		}
		return m, cmd
	case modeCreate:
		return m.updateCreate(msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	header := m.renderHeader()
	main := m.renderMain()
	footer := m.renderFooter()
	help := ""
	if m.showHelp {
		help = "\n" + m.renderHelp()
	}
	return header + "\n" + main + "\n" + footer + help
}

func (m model) renderHeader() string {
	status := "backend: " + m.status
	if !m.noColor && m.status == "ok" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	}
	return "Bountystash TUI " + version.Short() + " | " + status + " | " + m.baseURL
}

func (m model) renderMain() string {
	switch m.mode {
	case modeBrowse:
		return m.renderSplit("Browse", m.browseList.View())
	case modeReview:
		return m.renderSplit("Review", m.reviewList.View())
	case modeCreate:
		return m.renderCreate()
	default:
		return "unknown mode"
	}
}

func (m model) renderSplit(title string, left string) string {
	right := m.detail
	if strings.TrimSpace(right) == "" {
		right = "Select an item and press Enter to inspect."
	}

	leftWidth := max(28, m.width/3)
	rightWidth := max(40, m.width-leftWidth-3)
	leftPane := lipgloss.NewStyle().Width(leftWidth).Render(left)
	rightPane := lipgloss.NewStyle().Width(rightWidth).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " | ", rightPane)
}

func (m model) renderCreate() string {
	var b strings.Builder
	b.WriteString("Create Draft\n")
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(0, "Title"), m.titleInput.View()))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(1, "Kind"), kinds[m.kindIndex]))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(2, "Visibility"), visibilities[m.visibilityIndex]))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(3, "Reward"), m.rewardInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(4, "Scope"), m.scopeInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(5, "Deliverables"), m.delivInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(6, "Acceptance"), m.acceptInput.View()))
	b.WriteString("Submit: Ctrl+S")
	return b.String()
}

func (m model) focusLabel(idx int, name string) string {
	prefix := "  "
	if m.createFocus == idx {
		prefix = "> "
	}
	return prefix + name + ":"
}

func (m model) renderFooter() string {
	errText := m.errMsg
	if errText == "" && m.lastCreatedID != "" {
		errText = "created work item: " + m.lastCreatedID
	}
	if errText == "" {
		errText = "Press ? for help"
	}
	return fmt.Sprintf("mode=%s | %s", m.mode, errText)
}

func (m model) renderHelp() string {
	return "Keys: b browse | r review | c create | Enter inspect | Tab/Shift+Tab cycle create fields | Ctrl+S submit | Ctrl+L reload | q quit"
}

func (m *model) refreshBrowseItems() {
	items := make([]list.Item, 0, len(m.examples)+len(m.work))
	for _, ex := range m.examples {
		items = append(items, browseItem{
			title:  "example/" + ex.Slug,
			desc:   string(ex.Kind) + " · " + string(ex.Visibility),
			kind:   string(ex.Kind),
			source: "example",
			id:     ex.Slug,
		})
	}
	for _, w := range m.work {
		items = append(items, browseItem{
			title:  w.Title,
			desc:   w.ID + " · " + string(w.Kind) + " · " + string(w.Visibility),
			kind:   string(w.Kind),
			source: "work",
			id:     w.ID,
		})
	}
	m.browseList.SetItems(items)
}

func (m *model) refreshReviewItems() {
	items := make([]list.Item, 0, len(m.review.Standard)+len(m.review.Private))
	for _, row := range m.review.Standard {
		items = append(items, reviewItem{
			title: row.Title,
			desc:  row.ID + " · " + string(row.Kind) + " · " + row.Status,
			id:    row.ID,
		})
	}
	for _, row := range m.review.Private {
		items = append(items, reviewItem{
			title:       "[private_security] " + row.Title,
			desc:        row.ID + " · " + string(row.Kind) + " · " + row.Status,
			id:          row.ID,
			isSensitive: true,
		})
	}
	m.reviewList.SetItems(items)
}

func (m model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "down":
			m.setCreateFocus((m.createFocus + 1) % 7)
			return m, nil
		case "shift+tab", "up":
			m.setCreateFocus((m.createFocus + 6) % 7)
			return m, nil
		case "left":
			if m.createFocus == 1 {
				m.kindIndex = (m.kindIndex + len(kinds) - 1) % len(kinds)
				return m, nil
			}
			if m.createFocus == 2 {
				m.visibilityIndex = (m.visibilityIndex + len(visibilities) - 1) % len(visibilities)
				return m, nil
			}
		case "right":
			if m.createFocus == 1 {
				m.kindIndex = (m.kindIndex + 1) % len(kinds)
				return m, nil
			}
			if m.createFocus == 2 {
				m.visibilityIndex = (m.visibilityIndex + 1) % len(visibilities)
				return m, nil
			}
		case "ctrl+s":
			input := packets.DraftInput{
				Title:              m.titleInput.Value(),
				Kind:               kinds[m.kindIndex],
				Scope:              m.scopeInput.Value(),
				Deliverables:       m.delivInput.Value(),
				AcceptanceCriteria: m.acceptInput.Value(),
				RewardModel:        m.rewardInput.Value(),
				Visibility:         visibilities[m.visibilityIndex],
			}
			return m, createDraftCmd(m.api, input)
		}
	}

	var cmd tea.Cmd
	switch m.createFocus {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 3:
		m.rewardInput, cmd = m.rewardInput.Update(msg)
	case 4:
		m.scopeInput, cmd = m.scopeInput.Update(msg)
	case 5:
		m.delivInput, cmd = m.delivInput.Update(msg)
	case 6:
		m.acceptInput, cmd = m.acceptInput.Update(msg)
	}
	return m, cmd
}

func (m *model) setCreateFocus(idx int) {
	m.createFocus = idx
	if idx == 0 {
		m.titleInput.Focus()
	} else {
		m.titleInput.Blur()
	}
	if idx == 3 {
		m.rewardInput.Focus()
	} else {
		m.rewardInput.Blur()
	}
	if idx == 4 {
		m.scopeInput.Focus()
	} else {
		m.scopeInput.Blur()
	}
	if idx == 5 {
		m.delivInput.Focus()
	} else {
		m.delivInput.Blur()
	}
	if idx == 6 {
		m.acceptInput.Focus()
	} else {
		m.acceptInput.Blur()
	}
}

func (m *model) clearCreateForm() {
	m.titleInput.SetValue("")
	m.rewardInput.SetValue("")
	m.scopeInput.SetValue("")
	m.delivInput.SetValue("")
	m.acceptInput.SetValue("")
	m.kindIndex = 0
	m.visibilityIndex = 0
	m.createFocus = 0
	m.setCreateFocus(0)
}

func fetchHealthCmd(api clientapi.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := api.Health(context.Background())
		return healthMsg{status: status, err: err}
	}
}

func fetchExamplesCmd(api clientapi.Client) tea.Cmd {
	return func() tea.Msg {
		items, err := api.ListExamples(context.Background())
		return examplesMsg{items: items, err: err}
	}
}

func fetchWorkCmd(api clientapi.Client) tea.Cmd {
	return func() tea.Msg {
		items, err := api.ListWork(context.Background(), 25)
		return workListMsg{items: items, err: err}
	}
}

func fetchReviewCmd(api clientapi.Client) tea.Cmd {
	return func() tea.Msg {
		queue, err := api.ListReview(context.Background())
		return reviewMsg{queue: queue, err: err}
	}
}

func fetchExampleDetailCmd(api clientapi.Client, slug string) tea.Cmd {
	return func() tea.Msg {
		example, err := api.GetExample(context.Background(), slug)
		if err != nil {
			return detailMsg{err: err}
		}
		return detailMsg{text: formatExampleDetail(example)}
	}
}

func fetchWorkDetailCmd(api clientapi.Client, id string) tea.Cmd {
	return func() tea.Msg {
		work, err := api.GetWork(context.Background(), id)
		if err != nil {
			return detailMsg{err: err}
		}
		return detailMsg{text: formatWorkDetail(work)}
	}
}

func createDraftCmd(api clientapi.Client, input packets.DraftInput) tea.Cmd {
	return func() tea.Msg {
		result, err := api.CreateDraft(context.Background(), input)
		return draftCreateMsg{result: result, err: err}
	}
}

func formatExampleDetail(ex handlers.Example) string {
	return "Example: " + ex.Slug + "\n\n" + formatPacket(ex.Packet)
}

func formatWorkDetail(work handlers.WorkDetail) string {
	return strings.Join([]string{
		"Work Item: " + work.ID,
		"Status: " + work.Status,
		"Version: " + strconv.Itoa(work.Version),
		"Created: " + work.CreatedAt.Format(time.RFC3339),
		"Exact Hash: " + work.ExactHash,
		"Quotient Hash: " + work.QuotientHash,
		"",
		formatPacket(work.Packet),
	}, "\n")
}

func formatPacket(p packets.NormalizedPacket) string {
	return strings.Join([]string{
		"Title: " + p.Title,
		"Kind: " + string(p.Kind),
		"Visibility: " + string(p.Visibility),
		"Reward: " + p.RewardModel,
		"",
		"Scope:\n" + bulletLines(p.Scope),
		"",
		"Deliverables:\n" + bulletLines(p.Deliverables),
		"",
		"Acceptance:\n" + bulletLines(p.AcceptanceCriteria),
	}, "\n")
}

func bulletLines(lines []string) string {
	if len(lines) == 0 {
		return "(none)"
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, " - "+line)
	}
	return strings.Join(out, "\n")
}

func formatValidationErrors(errs packets.ValidationErrors) string {
	if len(errs) == 0 {
		return ""
	}
	lines := make([]string, 0, len(errs))
	for field, msg := range errs {
		lines = append(lines, field+": "+msg)
	}
	return strings.Join(lines, " | ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func resolveBaseURL(flagValue string) string {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(os.Getenv("BOUNTYSTASH_BASE_URL")); trimmed != "" {
		return trimmed
	}
	return compiledDefaultBaseURL
}

func main() {
	var (
		baseURL = flag.String("base-url", "", "Bountystash backend base URL")
		timeout = flag.Duration("timeout", 8*time.Second, "HTTP timeout")
		noColor = flag.Bool("no-color", false, "disable color styling")
		showVer = flag.Bool("version", false, "print version/build info and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println(version.BuildInfo())
		return
	}

	resolvedBaseURL := resolveBaseURL(*baseURL)

	api, err := clientapi.New(resolvedBaseURL, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize API client: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(api, resolvedBaseURL, *noColor), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
