package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	neturl "net/url"
	"os"
	"sort"
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
	modeBrowse  mode = "browse"
	modeReview  mode = "review"
	modeCreate  mode = "create"
	modeInspect mode = "inspect"
)

type browseItem struct {
	title      string
	desc       string
	kind       string
	source     string
	id         string
	selectable bool
}

func (i browseItem) Title() string       { return i.title }
func (i browseItem) Description() string { return i.desc }
func (i browseItem) FilterValue() string { return i.title + " " + i.desc + " " + i.kind }

type reviewItem struct {
	title      string
	desc       string
	id         string
	selectable bool
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
	title string
	text  string
	err   error
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
	prevMode mode
	showHelp bool

	status         string
	statusMsg      string
	errMsg         string
	pendingRefresh int

	width  int
	height int

	examples []clientapi.ExampleSummary
	work     []handlers.WorkListRow
	review   clientapi.ReviewQueue

	browseList  list.Model
	reviewList  list.Model
	detailTitle string
	detail      string

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
		prevMode:        modeBrowse,
		status:          "loading",
		statusMsg:       "initializing",
		pendingRefresh:  4,
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
		case "esc":
			if m.mode == modeInspect {
				m.mode = m.prevMode
				return m, nil
			}
			if m.mode == modeCreate {
				m.mode = modeBrowse
				return m, nil
			}
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+l":
			m.statusMsg = "refreshing"
			m.pendingRefresh = 4
			return m, tea.Batch(fetchHealthCmd(m.api), fetchExamplesCmd(m.api), fetchWorkCmd(m.api), fetchReviewCmd(m.api))
		}
	}

	switch msg := msg.(type) {
	case healthMsg:
		m.consumeRefresh()
		if msg.err != nil {
			m.status = "down"
			m.setError("health", msg.err)
		} else {
			m.status = msg.status
			m.statusMsg = "health check ok"
		}
		return m, nil
	case examplesMsg:
		m.consumeRefresh()
		if msg.err != nil {
			m.setError("examples", msg.err)
			return m, nil
		}
		m.examples = msg.items
		m.refreshBrowseItems()
		m.statusMsg = "examples loaded"
		return m, nil
	case workListMsg:
		m.consumeRefresh()
		if msg.err != nil {
			m.setError("work list", msg.err)
			return m, nil
		}
		m.work = msg.items
		m.refreshBrowseItems()
		m.statusMsg = "work list loaded"
		return m, nil
	case reviewMsg:
		m.consumeRefresh()
		if msg.err != nil {
			m.setError("review queue", msg.err)
			return m, nil
		}
		m.review = msg.queue
		m.refreshReviewItems()
		m.statusMsg = "review queue loaded"
		return m, nil
	case detailMsg:
		if msg.err != nil {
			m.setError("inspect", msg.err)
			return m, nil
		}
		m.detailTitle = msg.title
		m.detail = msg.text
		m.statusMsg = "inspect ready"
		m.errMsg = ""
		return m, nil
	case draftCreateMsg:
		if msg.err != nil {
			if apiErr, ok := msg.err.(*clientapi.APIError); ok && len(apiErr.ValidationErrors) > 0 {
				m.errMsg = formatValidationErrors(apiErr.ValidationErrors)
			} else {
				m.setError("create draft", msg.err)
			}
			return m, nil
		}
		m.lastCreatedID = msg.result.ID
		m.errMsg = ""
		m.detailTitle = "Created Work Item"
		m.detail = formatWorkDetail(handlers.WorkDetail{
			ID:           msg.result.ID,
			Status:       msg.result.Status,
			Version:      msg.result.Version,
			ExactHash:    msg.result.ExactHash,
			QuotientHash: msg.result.QuotientHash,
			Packet:       msg.result.Packet,
		})
		m.prevMode = modeBrowse
		m.mode = modeInspect
		m.statusMsg = "draft created"
		m.clearCreateForm()
		return m, tea.Batch(fetchWorkCmd(m.api), fetchReviewCmd(m.api))
	}

	switch m.mode {
	case modeBrowse:
		updated, cmd := m.browseList.Update(msg)
		m.browseList = updated
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			item, ok := m.browseList.SelectedItem().(browseItem)
			if !ok || !item.selectable {
				m.statusMsg = "select a work item or example to inspect"
				return m, nil
			}
			m.prevMode = modeBrowse
			m.mode = modeInspect
			m.detailTitle = "Loading Inspect View"
			m.detail = "Loading details..."
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
			if !ok || !item.selectable {
				m.statusMsg = "select a review item to inspect"
				return m, nil
			}
			m.prevMode = modeReview
			m.mode = modeInspect
			m.detailTitle = "Loading Inspect View"
			m.detail = "Loading details..."
			return m, fetchWorkDetailCmd(m.api, item.id)
		}
		return m, cmd
	case modeCreate:
		return m.updateCreate(msg)
	case modeInspect:
		return m, nil
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
	status := "health: " + m.status
	if !m.noColor && m.status == "ok" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	} else if !m.noColor && m.status == "down" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status)
	}
	return "Bountystash TUI " + version.Short() + " | backend: " + m.baseURL + " | " + status + " | mode: " + string(m.mode)
}

func (m model) renderMain() string {
	switch m.mode {
	case modeBrowse:
		return m.renderSplit("Browse", m.browseList.View())
	case modeReview:
		return m.renderSplit("Review", m.reviewList.View())
	case modeCreate:
		return m.renderCreate()
	case modeInspect:
		return m.renderInspect()
	default:
		return "unknown mode"
	}
}

func (m model) renderSplit(title string, left string) string {
	if title == "Browse" && len(m.browseList.Items()) == 0 {
		left = left + "\n\nNo examples or work items available."
	}
	if title == "Review" && len(m.reviewList.Items()) == 0 {
		left = left + "\n\nReview queue is empty."
	}

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

func (m model) renderInspect() string {
	if strings.TrimSpace(m.detail) == "" {
		return "Inspect\n\nNo detail loaded yet."
	}

	var b strings.Builder
	title := strings.TrimSpace(m.detailTitle)
	if title == "" {
		title = "Inspect"
	}
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", max(10, min(60, m.width-2))))
	b.WriteString("\n")
	b.WriteString(m.detail)
	b.WriteString("\n\nEsc: back")
	return b.String()
}

func (m model) renderCreate() string {
	var b strings.Builder
	b.WriteString("Create Draft (Ctrl+S submit, Esc back)\n")
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(0, "Title"), m.titleInput.View()))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(1, "Kind"), kinds[m.kindIndex]))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(2, "Visibility"), visibilities[m.visibilityIndex]))
	b.WriteString(fmt.Sprintf("%s %s\n", m.focusLabel(3, "Reward Model"), m.rewardInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(4, "Scope"), m.scopeInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(5, "Deliverables"), m.delivInput.View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n", m.focusLabel(6, "Acceptance Criteria"), m.acceptInput.View()))
	b.WriteString("Tab/Shift+Tab: focus field | Left/Right: cycle kind and visibility")
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
	if m.errMsg != "" {
		return fmt.Sprintf("mode=%s | ERROR: %s", m.mode, m.errMsg)
	}
	status := m.statusMsg
	if status == "" && m.lastCreatedID != "" {
		status = "created work item: " + m.lastCreatedID
	}
	if status == "" {
		status = "ready"
	}
	if m.pendingRefresh > 0 {
		status = status + fmt.Sprintf(" | loading: %d", m.pendingRefresh)
	}
	return fmt.Sprintf("mode=%s | %s | keys: ? help", m.mode, status)
}

func (m model) renderHelp() string {
	return strings.Join([]string{
		"Global: b browse | r review | c create | ? help | Ctrl+L reload | q quit",
		"Browse/Review: Up/Down move | Enter inspect selected item",
		"Inspect: Esc back",
		"Create: Tab/Shift+Tab focus | Left/Right cycle kind/visibility | Ctrl+S submit | Esc back",
	}, "\n")
}

func (m *model) refreshBrowseItems() {
	items := make([]list.Item, 0, len(m.examples)+len(m.work)+2)
	items = append(items, browseItem{title: "Examples", source: "section", selectable: false})
	for _, ex := range m.examples {
		items = append(items, browseItem{
			title:      "[example] " + ex.Title,
			desc:       ex.Slug + " · " + string(ex.Kind) + " · " + string(ex.Visibility),
			kind:       string(ex.Kind),
			source:     "example",
			id:         ex.Slug,
			selectable: true,
		})
	}
	items = append(items, browseItem{title: "Recent Work Items", source: "section", selectable: false})
	for _, w := range m.work {
		items = append(items, browseItem{
			title:      "[work] " + w.Title,
			desc:       w.ID + " · " + string(w.Kind) + " · " + string(w.Visibility) + " · " + w.Status,
			kind:       string(w.Kind),
			source:     "work",
			id:         w.ID,
			selectable: true,
		})
	}
	m.browseList.Title = fmt.Sprintf("Browse (%d examples, %d work)", len(m.examples), len(m.work))
	m.browseList.SetItems(items)
}

func (m *model) refreshReviewItems() {
	items := make([]list.Item, 0, len(m.review.Standard)+len(m.review.Private)+2)
	items = append(items, reviewItem{title: "Standard Review Queue", selectable: false})
	for _, row := range m.review.Standard {
		items = append(items, reviewItem{
			title:      row.Title,
			desc:       row.ID + " · " + string(row.Kind) + " · " + row.Status,
			id:         row.ID,
			selectable: true,
		})
	}
	items = append(items, reviewItem{title: "Private Security Queue", selectable: false})
	for _, row := range m.review.Private {
		items = append(items, reviewItem{
			title:      "[private_security] " + row.Title,
			desc:       row.ID + " · " + string(row.Kind) + " · " + row.Status + " · restricted",
			id:         row.ID,
			selectable: true,
		})
	}
	m.reviewList.Title = fmt.Sprintf("Review (%d standard, %d private)", len(m.review.Standard), len(m.review.Private))
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
			m.statusMsg = "submitting draft"
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
		return detailMsg{
			title: "Example: " + example.Slug,
			text:  formatExampleDetail(example),
		}
	}
}

func fetchWorkDetailCmd(api clientapi.Client, id string) tea.Cmd {
	return func() tea.Msg {
		work, err := api.GetWork(context.Background(), id)
		if err != nil {
			return detailMsg{err: err}
		}
		return detailMsg{
			title: "Work Item: " + work.ID,
			text:  formatWorkDetail(work),
		}
	}
}

func createDraftCmd(api clientapi.Client, input packets.DraftInput) tea.Cmd {
	return func() tea.Msg {
		result, err := api.CreateDraft(context.Background(), input)
		return draftCreateMsg{result: result, err: err}
	}
}

func formatExampleDetail(ex handlers.Example) string {
	return strings.Join([]string{
		"Example Slug: " + ex.Slug,
		"",
		formatPacket(ex.Packet),
	}, "\n")
}

func formatWorkDetail(work handlers.WorkDetail) string {
	created := "(not available)"
	if !work.CreatedAt.IsZero() {
		created = work.CreatedAt.Format(time.RFC3339)
	}
	return strings.Join([]string{
		"Work Item ID: " + work.ID,
		"Status: " + work.Status,
		"Version: " + strconv.Itoa(work.Version),
		"Created At: " + created,
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
		"Reward Model: " + p.RewardModel,
		"",
		"Scope",
		bulletLines(p.Scope),
		"",
		"Deliverables",
		bulletLines(p.Deliverables),
		"",
		"Acceptance Criteria",
		bulletLines(p.AcceptanceCriteria),
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
	keys := make([]string, 0, len(errs))
	for field := range errs {
		keys = append(keys, field)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, field := range keys {
		lines = append(lines, field+": "+errs[field])
	}
	return "validation failed: " + strings.Join(lines, " | ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *model) consumeRefresh() {
	if m.pendingRefresh <= 0 {
		return
	}
	m.pendingRefresh--
	if m.pendingRefresh == 0 {
		m.statusMsg = "refresh complete"
	}
}

func (m *model) setError(contextLabel string, err error) {
	m.errMsg = contextLabel + ": " + describeError(err)
}

func describeError(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *clientapi.APIError
	if errors.As(err, &apiErr) {
		if len(apiErr.ValidationErrors) > 0 {
			return formatValidationErrors(apiErr.ValidationErrors)
		}
		return fmt.Sprintf("api error (%d): %s", apiErr.StatusCode, apiErr.Message)
	}
	var urlErr *neturl.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return "request timeout; backend may be unavailable"
		}
		return "backend unavailable: " + urlErr.Err.Error()
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "request timeout; backend may be unavailable"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timeout; backend may be unavailable"
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "invalid json") {
		return "invalid response from backend"
	}
	return msg
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
