package report

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alesr/impact/internal/estimate"
)

func PrintJSON(rep estimate.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(rep); err != nil {
		return fmt.Errorf("could not encode json report: %w", err)
	}
	return nil
}
