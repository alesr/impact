package report

import (
	"fmt"
	"os"

	"github.com/alesr/impact/internal/estimate"
	"github.com/alesr/impact/internal/pkg/planview"
	"github.com/jedib0t/go-pretty/v6/table"
)

func PrintTable(rep estimate.Report) error {
	fmt.Fprintf(os.Stdout, "Totals\n")
	fmt.Fprintf(os.Stdout, "  kgCO2e/month: %s\n", planview.FormatKg(rep.Totals.KgCO2eMonth, rep.Totals.KgCO2eKnown))
	fmt.Fprintf(os.Stdout, "  m3 water/month: %s\n", planview.FormatWater(rep.Totals.M3WaterMonth, rep.Totals.M3WaterKnown))
	if note := planview.UnknownImpactNote(rep.Totals.UnknownRows); note != "" {
		fmt.Fprintf(os.Stdout, "  note: %s\n", note)
	}
	fmt.Fprintf(os.Stdout, "\n")

	planview.SortRows(rep.Rows, planview.SortByCO2)

	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"ADDRESS", "ACTION", "KGCO2E/MO", "M3/MO", "SKU"})

	for _, row := range rep.Rows {
		tw.AppendRow(table.Row{row.Address, row.Action, planview.FormatKg(row.KgCO2eMonth, row.KgCO2eKnown), planview.FormatWater(row.M3WaterMonth, row.M3WaterKnown), row.SKU})
	}

	tw.Render()

	if len(rep.Unsupported) > 0 {
		fmt.Fprintf(os.Stdout, "\nUnsupported resources (%d):\n", len(rep.Unsupported))
		for _, unsupported := range rep.Unsupported {
			fmt.Fprintf(os.Stdout, "  - %s: %s\n", unsupported.Address, unsupported.Reason)
		}
	}
	return nil
}
