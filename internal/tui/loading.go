package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alesr/impact/internal/estimate"
	"github.com/alesr/impact/internal/pkg/progress"
)

type buildPlanReportFn func() (estimate.Report, error)

type loadingDoneMsg struct {
	report estimate.Report
	err    error
}

type loadingModel struct {
	buildFn  buildPlanReportFn
	spinner  spinner.Model
	width    int
	height   int
	started  time.Time
	finished bool
}

type loadingErrorModel struct {
	err error
}

func RunPlanReportLoading(buildFn buildPlanReportFn) error {
	spin := spinner.New(spinner.WithSpinner(progress.DotSpinner()))
	spin.Style = subtleStyle

	m := loadingModel{
		buildFn: buildFn,
		spinner: spin,
		started: time.Now(),
		width:   120,
		height:  30,
	}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}

	if failed, ok := finalModel.(loadingErrorModel); ok {
		return failed.err
	}

	return nil
}

func (m loadingModel) Init() tea.Cmd {
	return tea.Batch(m.runBuild(), m.spinner.Tick)
}

func (m loadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		if m.finished {
			return m, nil
		}

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case loadingDoneMsg:
		if msg.err != nil {
			return loadingErrorModel{err: msg.err}, tea.Quit
		}

		reportModel := newPlanModel(msg.report)
		reportModel.width = m.width
		if m.height > 12 {
			reportModel.height = m.height - 10
			if reportModel.height < 5 {
				reportModel.height = 5
			}
		}

		m.finished = true
		return reportModel, nil
	}

	return m, nil
}

func (m loadingErrorModel) Init() tea.Cmd { return nil }

func (m loadingErrorModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m loadingErrorModel) View() string { return "" }

func (m loadingModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("impact"))
	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render("Terraform plan impact report"))
	b.WriteString("\n\n")
	b.WriteString(chipStyle.Render(fmt.Sprintf("%s processing plan and fetching catalog", m.spinner.View())))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(fmt.Sprintf("elapsed: %s", time.Since(m.started).Truncate(time.Second))))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("press q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m loadingModel) runBuild() tea.Cmd {
	return func() tea.Msg {
		rep, err := m.buildFn()
		return loadingDoneMsg{report: rep, err: err}
	}
}
