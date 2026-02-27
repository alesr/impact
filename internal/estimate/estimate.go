package estimate

import (
	"errors"
	"slices"
	"strings"

	"github.com/alesr/impact/internal/mapping"
	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/scw/catalog"
)

const monthlyHours = 730.0

type Row struct {
	Address      string  `json:"address"`
	Type         string  `json:"type"`
	Action       string  `json:"action"`
	SKU          string  `json:"sku,omitempty"`
	KgCO2eMonth  float64 `json:"kgco2e_month"`
	KgCO2eKnown  bool    `json:"kgco2e_known"`
	M3WaterMonth float64 `json:"m3_water_month"`
	M3WaterKnown bool    `json:"m3_water_known"`
}

type Report struct {
	Rows        []Row                 `json:"rows"`
	Unsupported []UnsupportedResource `json:"unsupported"`
	Totals      Totals                `json:"totals"`
}

type UnsupportedResource struct {
	Address string `json:"address"`
	Code    string `json:"code"`
	Reason  string `json:"reason"`
}

type Totals struct {
	KgCO2eMonth  float64 `json:"kgco2e_month"`
	KgCO2eKnown  bool    `json:"kgco2e_known"`
	M3WaterMonth float64 `json:"m3_water_month"`
	M3WaterKnown bool    `json:"m3_water_known"`
	UnknownRows  int     `json:"unknown_rows"`
}

func Build(changes []plan.ResourceChange, products []catalog.Product) Report {
	report := Report{
		Rows:        make([]Row, 0, len(changes)),
		Unsupported: []UnsupportedResource{},
	}

	var unknownKgRows, unknownWaterRows float64

	for _, change := range changes {
		transitions := actionTransitions(change)
		for _, transition := range transitions {
			match, err := mapping.Resolve(transition.Change, products)
			if err != nil || (match.Product == nil && len(match.Matches) == 0) {
				report.Unsupported = append(report.Unsupported, unsupportedFromError(change.Address, err))
				continue
			}

			rows := rowsFromMatch(transition.Change, transition.Action, transition.Multiplier, match)
			for _, row := range rows {
				report.Rows = append(report.Rows, row)
				if row.KgCO2eKnown {
					report.Totals.KgCO2eMonth += row.KgCO2eMonth
				}
				if row.M3WaterKnown {
					report.Totals.M3WaterMonth += row.M3WaterMonth
				}
				if !row.KgCO2eKnown || !row.M3WaterKnown {
					report.Totals.UnknownRows++
				}
				if !row.KgCO2eKnown {
					unknownKgRows++
				}
				if !row.M3WaterKnown {
					unknownWaterRows++
				}
			}
		}
	}

	report.Totals.KgCO2eKnown = unknownKgRows == 0
	report.Totals.M3WaterKnown = unknownWaterRows == 0

	return report
}

func rowsFromMatch(change plan.ResourceChange, action string, multiplier float64, match mapping.Result) []Row {
	if len(match.Matches) == 0 {
		return []Row{rowFromProduct(change, action, multiplier, match.Qty, *match.Product)}
	}

	rows := make([]Row, 0, len(match.Matches))
	for _, m := range match.Matches {
		rows = append(rows, rowFromProduct(change, action, multiplier, m.Qty, m.Product))
	}
	return rows
}

func unsupportedFromError(address string, err error) UnsupportedResource {
	unsupported := UnsupportedResource{
		Address: address,
		Code:    string(mapping.ErrorCodeNoCatalogMatch),
		Reason:  "no matching catalog product",
	}

	var mappingErr *mapping.Error
	if errors.As(err, &mappingErr) {
		unsupported.Code = string(mappingErr.Code)
		unsupported.Reason = mappingErr.Reason
	}
	return unsupported
}

func rowFromProduct(change plan.ResourceChange, action string, multiplier float64, qty float64, product catalog.Product) Row {
	billedQty := normalizeQtyByUnitSize(qty, product.UnitOfMeasure.Size)

	var (
		kg, m3           float64
		kgKnown, m3Known bool
	)

	if product.EnvironmentalImpactEstimation != nil {
		if product.EnvironmentalImpactEstimation.KgCO2Equivalent != nil {
			kg = *product.EnvironmentalImpactEstimation.KgCO2Equivalent
			kgKnown = true
		}
		if product.EnvironmentalImpactEstimation.M3WaterUsage != nil {
			m3 = *product.EnvironmentalImpactEstimation.M3WaterUsage
			m3Known = true
		}
	}

	unitMultiplier := unitToMonthMultiplier(product.UnitOfMeasure.Unit)

	return Row{
		Address:      change.Address,
		Type:         change.Type,
		Action:       action,
		SKU:          product.SKU,
		KgCO2eMonth:  kg * billedQty * unitMultiplier * multiplier,
		KgCO2eKnown:  kgKnown,
		M3WaterMonth: m3 * billedQty * unitMultiplier * multiplier,
		M3WaterKnown: m3Known,
	}
}

type actionTransition struct {
	Change     plan.ResourceChange
	Action     string
	Multiplier float64
}

func actionTransitions(change plan.ResourceChange) []actionTransition {
	hasCreate := slices.Contains(change.Actions, "create")
	hasDelete := slices.Contains(change.Actions, "delete")
	hasUpdate := slices.Contains(change.Actions, "update")

	transitions := make([]actionTransition, 0, 2)

	if hasDelete {
		if before := beforeChange(change); hasBeforeData(before) {
			transitions = append(transitions, actionTransition{Change: before, Action: "delete", Multiplier: -1})
		}
	}

	if hasCreate {
		if after := afterChange(change); hasAfterData(after) {
			transitions = append(transitions, actionTransition{Change: after, Action: "create", Multiplier: 1})
		}
	}

	if len(transitions) > 0 {
		return transitions
	}

	if hasUpdate {
		if before := beforeChange(change); hasBeforeData(before) {
			transitions = append(transitions, actionTransition{Change: before, Action: "update", Multiplier: -1})
		}
		if after := afterChange(change); hasAfterData(after) {
			transitions = append(transitions, actionTransition{Change: after, Action: "update", Multiplier: 1})
		}
	}

	if len(transitions) == 0 {
		return nil
	}
	return transitions
}

func beforeChange(change plan.ResourceChange) plan.ResourceChange {
	before := change
	before.After = nil
	return before
}

func afterChange(change plan.ResourceChange) plan.ResourceChange {
	after := change
	after.Before = nil
	return after
}

func hasBeforeData(change plan.ResourceChange) bool {
	return len(change.Before) > 0
}

func hasAfterData(change plan.ResourceChange) bool {
	return len(change.After) > 0
}

func normalizeQtyByUnitSize(qty float64, size uint64) float64 {
	if size == 0 || size == 1 {
		return qty
	}

	return qty / float64(size)
}

func unitToMonthMultiplier(unit string) float64 {
	switch strings.ToLower(unit) {
	case "hour":
		return monthlyHours
	case "month":
		return 1
	case "year":
		return 1.0 / 12.0
	default:
		return 1
	}
}
