package strx

import "strings"

func ParseCSV(values ...string) []string {
	out := make([]string, 0, len(values)*2)

	for _, value := range values {
		parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' })
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
