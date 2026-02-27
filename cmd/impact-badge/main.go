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
	Totals struct {
		KgCO2eMonth  float64 `json:"kgco2e_month"`
		M3WaterMonth float64 `json:"m3_water_month"`
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

	b.Message = fmt.Sprintf("~%s kgCO2e | ~%s m3", formatValue(rep.Totals.KgCO2eMonth), formatValue(rep.Totals.M3WaterMonth))
	b.Color = "2e7d32"
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
