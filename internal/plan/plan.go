package plan

import (
	"fmt"
	"os"

	tfjson "github.com/hashicorp/terraform-json"
)

const maxPlanFileBytes = 50 << 20

type ResourceChange struct {
	Address string
	Type    string
	Actions []string
	Before  map[string]any
	After   map[string]any
	Zone    string
	Region  string
}

func ParseFile(filePath string) ([]ResourceChange, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read plan file: %w", err)
	}
	if info.Size() > maxPlanFileBytes {
		return nil, fmt.Errorf("could not read plan file: file too large (%d bytes > %d bytes)", info.Size(), maxPlanFileBytes)
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read plan file: %w", err)
	}
	return ParseBytes(b)
}

func ParseBytes(data []byte) ([]ResourceChange, error) {
	if len(data) > maxPlanFileBytes {
		return nil, fmt.Errorf("could not decode terraform plan json: payload too large (%d bytes > %d bytes)", len(data), maxPlanFileBytes)
	}

	plan := new(tfjson.Plan)
	if err := plan.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("could not decode terraform plan json: %w", err)
	}

	changes := make([]ResourceChange, 0, len(plan.ResourceChanges))
	defaultZone := planVariableString(plan, "zone")
	defaultRegion := planVariableString(plan, "region")

	for _, rc := range plan.ResourceChanges {
		if rc == nil {
			continue
		}

		before := map[string]any{}
		after := map[string]any{}
		actions := []string{}

		if rc.Change != nil {
			before = anyToMap(rc.Change.Before)
			after = anyToMap(rc.Change.After)
			for _, action := range rc.Change.Actions {
				actions = append(actions, string(action))
			}
		}

		changes = append(changes, ResourceChange{
			Address: rc.Address,
			Type:    rc.Type,
			Actions: actions,
			Before:  before,
			After:   after,
			Zone:    defaultZone,
			Region:  defaultRegion,
		})
	}
	return changes, nil
}

func anyToMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return map[string]any{}
	}
	return m
}

func planVariableString(plan *tfjson.Plan, key string) string {
	if plan == nil || plan.Variables == nil {
		return ""
	}

	v, ok := plan.Variables[key]
	if !ok || v == nil || v.Value == nil {
		return ""
	}

	s, ok := v.Value.(string)
	if !ok {
		return ""
	}
	return s
}
