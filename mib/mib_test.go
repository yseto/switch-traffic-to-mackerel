package mib

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestValidate(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		var all []string
		actual, err := Validate(all)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []string{
			"ifOutDiscards",
			"ifInErrors",
			"ifOutErrors",
			"ifHCInOctets",
			"ifHCOutOctets",
			"ifInDiscards",
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j string) bool { return i < j }),
		); d != "" {
			t.Errorf("invalid results %s", d)
		}
	})

	t.Run("some values", func(t *testing.T) {
		v := []string{"ifInErrors", "ifHCInOctets"}
		actual, err := Validate(v)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []string{
			"ifInErrors",
			"ifHCInOctets",
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j string) bool { return i < j }),
		); d != "" {
			t.Errorf("invalid results %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		v := []string{"aaaaaaaaaaaa"}
		result, err := Validate(v)
		if err == nil {
			t.Error("failed raised error")
		}
		if result != nil {
			t.Errorf("invalid result")
		}
	})

}
