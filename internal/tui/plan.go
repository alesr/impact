package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alesr/impact/internal/estimate"
	"github.com/alesr/impact/internal/pkg/planview"
)

type tab int

const (
	tabRows tab = iota
	tabUnsupported
)

type sortMode int

const (
	sortByCO2 sortMode = iota
	sortByWater
)

type planModel struct {
	report            estimate.Report
	rows              []estimate.Row
	unsupported       []estimate.UnsupportedResource
	tab               tab
	sortMode          sortMode
	cursorRows        int
	cursorUnsupported int
	offsetRows        int
	offsetUnsupported int
	height            int
	width             int
}

func newPlanModel(report estimate.Report) planModel {
	rows := make([]estimate.Row, len(report.Rows))
	copy(rows, report.Rows)

	unsupported := make([]estimate.UnsupportedResource, len(report.Unsupported))
	copy(unsupported, report.Unsupported)

	m := planModel{
		report:      report,
		rows:        rows,
		unsupported: unsupported,
		sortMode:    sortByCO2,
		height:      12,
		width:       120,
	}
	m.sortRows()

	return m
}

func (m planModel) Init() tea.Cmd { return nil }

func (m planModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 12 {
			m.height = msg.Height - 10
		}
		if m.height < 5 {
			m.height = 5
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "l", "right", "shift+tab", "h", "left":
			m.tab = (m.tab + 1) % 2
		case "s":
			m.sortMode = (m.sortMode + 1) % 2
			m.sortRows()
			m.cursorRows, m.offsetRows = 0, 0
		case "1":
			m.tab = tabRows
		case "2":
			m.tab = tabUnsupported
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		}
	}

	m.clampOffsets()
	return m, nil
}

func (m planModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("impact"))
	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render("Terraform plan impact report"))
	b.WriteString("\n")

	b.WriteString(chipStyle.Render(fmt.Sprintf("kgCO2e/mo %s", planview.FormatKg(m.report.Totals.KgCO2eMonth, m.report.Totals.KgCO2eKnown))))
	b.WriteString(" ")
	b.WriteString(chipStyle.Render(fmt.Sprintf("m3/mo %s", planview.FormatWater(m.report.Totals.M3WaterMonth, m.report.Totals.M3WaterKnown))))
	b.WriteString("\n")
	if note := planview.UnknownImpactNote(m.report.Totals.UnknownRows); note != "" {
		b.WriteString(subtleStyle.Render(note))
		b.WriteString("\n")
	}

	b.WriteString(tabStyle.Render(m.tabLabel(tabRows)))
	b.WriteString(" ")
	b.WriteString(tabStyle.Render(m.tabLabel(tabUnsupported)))
	b.WriteString("   ")
	b.WriteString(subtleStyle.Render("Sort: " + m.sortLabel()))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Keys: ↑/↓ (j/k) move  ←/→ (h/l) switch tab  s sort  q quit"))
	b.WriteString("\n\n")

	switch m.tab {
	case tabRows:
		addrWidth := 42
		if m.width > 0 && m.width < 120 {
			addrWidth = 28
		}
		head := fmt.Sprintf("  %-*s %-8s %12s %10s", addrWidth, "Address", "Action", "kgCO2e/mo", "m3/mo")
		b.WriteString(headerStyle.Render(head))
		b.WriteString("\n")

		if len(m.rows) == 0 {
			b.WriteString(subtleStyle.Render("  no supported resources in plan"))
			b.WriteString("\n")
			break
		}

		end := m.offsetRows + m.height
		if end > len(m.rows) {
			end = len(m.rows)
		}
		for i := m.offsetRows; i < end; i++ {
			row := m.rows[i]
			prefix := " "
			if i == m.cursorRows {
				prefix = ">"
			}
			line := fmt.Sprintf(
				"%s %-*s %-8s %12s %10s",
				prefix,
				addrWidth,
				truncate(row.Address, addrWidth),
				row.Action,
				planview.FormatKg(row.KgCO2eMonth, row.KgCO2eKnown),
				planview.FormatWater(row.M3WaterMonth, row.M3WaterKnown),
			)
			if i == m.cursorRows {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}

		if len(m.rows) > 0 && m.cursorRows < len(m.rows) {
			selected := m.rows[m.cursorRows]
			detail := strings.Builder{}
			detail.WriteString(fmt.Sprintf("Selected: %s\n", selected.Address))
			detail.WriteString(fmt.Sprintf("SKU: %s", selected.SKU))
			b.WriteString("\n")
			b.WriteString(detailStyle.Render(detail.String()))
		}

	case tabUnsupported:
		b.WriteString(headerStyle.Render(fmt.Sprintf("Unsupported resources (%d)", len(m.unsupported))))
		b.WriteString("\n")
		if len(m.unsupported) == 0 {
			b.WriteString(subtleStyle.Render("  none"))
			b.WriteString("\n")
			break
		}

		end := m.offsetUnsupported + m.height
		if end > len(m.unsupported) {
			end = len(m.unsupported)
		}
		for i := m.offsetUnsupported; i < end; i++ {
			prefix := " "
			if i == m.cursorUnsupported {
				prefix = ">"
			}
			line := fmt.Sprintf("%s %s: %s", prefix, m.unsupported[i].Address, m.unsupported[i].Reason)
			if i == m.cursorUnsupported {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m *planModel) sortRows() {
	sortKey := planview.SortByCO2
	switch m.sortMode {
	case sortByCO2:
		sortKey = planview.SortByCO2
	case sortByWater:
		sortKey = planview.SortByWater
	}

	planview.SortRows(m.rows, sortKey)
}

func (m planModel) sortLabel() string {
	switch m.sortMode {
	case sortByCO2:
		return "kgCO2e"
	case sortByWater:
		return "m3 water"
	default:
		return "kgCO2e"
	}
}

func (m planModel) tabLabel(t tab) string {
	name := "rows"
	if t == tabUnsupported {
		name = "unsupported"
	}
	if m.tab == t {
		return strings.ToUpper(name)
	}
	return name
}

func (m *planModel) moveCursor(delta int) {
	switch m.tab {
	case tabRows:
		m.cursorRows += delta
		if m.cursorRows < 0 {
			m.cursorRows = 0
		}
		if m.cursorRows >= len(m.rows) && len(m.rows) > 0 {
			m.cursorRows = len(m.rows) - 1
		}
	case tabUnsupported:
		m.cursorUnsupported += delta
		if m.cursorUnsupported < 0 {
			m.cursorUnsupported = 0
		}
		if m.cursorUnsupported >= len(m.unsupported) && len(m.unsupported) > 0 {
			m.cursorUnsupported = len(m.unsupported) - 1
		}
	}
}

func (m *planModel) clampOffsets() {
	clamp := func(cursor *int, offset *int, size int, height int) {
		if size <= 0 {
			*cursor = 0
			*offset = 0
			return
		}
		if *cursor < *offset {
			*offset = *cursor
		}
		if *cursor >= *offset+height {
			*offset = *cursor - height + 1
		}
		if *offset < 0 {
			*offset = 0
		}
	}

	clamp(&m.cursorRows, &m.offsetRows, len(m.rows), m.height)
	clamp(&m.cursorUnsupported, &m.offsetUnsupported, len(m.unsupported), m.height)
}

func truncate(v string, max int) string {
	if max <= 3 || len(v) <= max {
		return v
	}
	return v[:max-3] + "..."
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	chipStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1)
	tabStyle      = lipgloss.NewStyle().Bold(true)
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	detailStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
