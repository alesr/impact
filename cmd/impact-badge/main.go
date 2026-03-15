package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type impactReport struct {
	Unsupported []struct{} `json:"unsupported"`
	Totals      struct {
		KgCO2eMonth  float64 `json:"kgco2e_month"`
		KgCO2eKnown  bool    `json:"kgco2e_known"`
		M3WaterMonth float64 `json:"m3_water_month"`
		M3WaterKnown bool    `json:"m3_water_known"`
		UnknownRows  int     `json:"unknown_rows"`
	} `json:"totals"`
}

type badge struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
	CacheSeconds  int    `json:"cacheSeconds"`
}

func main() {
	input := flag.String("input", "", "Path to impact JSON (from `impact plan --format json`).")
	output := flag.String("output", "", "Path to write Shields endpoint JSON.")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "both --input and --output are required")
		os.Exit(2)
	}

	imp, err := readImpactReport(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read impact report: %v\n", err)
		os.Exit(1)
	}

	b := buildBadge(imp)
	if err := writeBadge(*output, b); err != nil {
		fmt.Fprintf(os.Stderr, "could not write badge: %v\n", err)
		os.Exit(1)
	}
}

func readImpactReport(path string) (*impactReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rep impactReport
	if err := json.NewDecoder(f).Decode(&rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func buildBadge(rep *impactReport) badge {
	b := badge{
		SchemaVersion: 1,
		Label:         "impact estimate/month",
		Message:       "estimate unavailable",
		Color:         "9e9e9e",
		CacheSeconds:  3600,
	}

	if rep == nil {
		return b
	}

	message := fmt.Sprintf("~%s kgCO2e | ~%s m3", formatValue(rep.Totals.KgCO2eMonth), formatValue(rep.Totals.M3WaterMonth))
	isComplete := rep.Totals.KgCO2eKnown && rep.Totals.M3WaterKnown && rep.Totals.UnknownRows == 0 && len(rep.Unsupported) == 0

	if isComplete {
		b.Message = message
		b.Color = "2e7d32"
		return b
	}

	b.Message = message + " (partial)"
	b.Color = "f9a825"
	return b
}

func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'g', 6, 64)
}

func writeBadge(path string, b badge) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}
