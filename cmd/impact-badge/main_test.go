package main

import "testing"

func TestBuildBadge(t *testing.T) {
	t.Parallel()

	t.Run("returns green badge for complete estimate", func(t *testing.T) {
		t.Parallel()

		rep := &impactReport{}
		rep.Totals.KgCO2eMonth = 12.34
		rep.Totals.M3WaterMonth = 0.56
		rep.Totals.KgCO2eKnown = true
		rep.Totals.M3WaterKnown = true

		b := buildBadge(rep)
		if b.Color != "2e7d32" {
			t.Fatalf("expected green badge color, got %q", b.Color)
		}
		if b.Message != "~12.34 kgCO2e | ~0.56 m3" {
			t.Fatalf("unexpected message: %q", b.Message)
		}
	})

	t.Run("returns amber badge for partial estimate", func(t *testing.T) {
		t.Parallel()

		rep := &impactReport{}
		rep.Totals.KgCO2eMonth = 0
		rep.Totals.M3WaterMonth = 0
		rep.Totals.KgCO2eKnown = false
		rep.Totals.M3WaterKnown = false
		rep.Totals.UnknownRows = 2
		rep.Unsupported = []struct{}{{}}

		b := buildBadge(rep)
		if b.Color != "f9a825" {
			t.Fatalf("expected amber badge color, got %q", b.Color)
		}
		if b.Message != "~0 kgCO2e | ~0 m3 (partial)" {
			t.Fatalf("unexpected message: %q", b.Message)
		}
	})
}
