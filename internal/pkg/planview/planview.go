package planview

import (
	"fmt"
	"sort"

	"github.com/alesr/impact/internal/estimate"
)

const na = "N/A"

type SortKey int

const (
	SortByCO2 SortKey = iota
	SortByWater
)

func FormatKg(v float64, known bool) string {
	if !known {
		return na
	}
	return fmt.Sprintf("%.6f", v)
}

func FormatWater(v float64, known bool) string {
	if !known {
		return na
	}
	return fmt.Sprintf("%.6f", v)
}

func UnknownImpactNote(unknownRows int) string {
	if unknownRows <= 0 {
		return ""
	}
	return fmt.Sprintf("partial totals: %d row(s) have unknown footprint data", unknownRows)
}

func SortRows(rows []estimate.Row, key SortKey) {
	sort.Slice(rows, func(i, j int) bool {
		a := rows[i]
		b := rows[j]

		switch key {
		case SortByCO2:
			return lessWithUnknownLast(a.KgCO2eMonth, a.KgCO2eKnown, b.KgCO2eMonth, b.KgCO2eKnown)
		case SortByWater:
			return lessWithUnknownLast(a.M3WaterMonth, a.M3WaterKnown, b.M3WaterMonth, b.M3WaterKnown)
		default:
			return lessWithUnknownLast(a.KgCO2eMonth, a.KgCO2eKnown, b.KgCO2eMonth, b.KgCO2eKnown)
		}
	})
}

func lessWithUnknownLast(aValue float64, aKnown bool, bValue float64, bKnown bool) bool {
	if aKnown != bKnown {
		return aKnown
	}
	if !aKnown {
		return false
	}
	return aValue > bValue
}
