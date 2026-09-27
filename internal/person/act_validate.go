package person

import (
	"fmt"
)

// validateActs requires every list, and every boundary side, to name at least
// one act. How many is a guideline in docs/kdl-contracts.md, not a check.
func validateActs(owner string, acts []Act, sided bool) error {
	if !sided {
		if len(acts) == 0 {
			return fmt.Errorf("%s names no acts", owner)
		}
		return validateActTexts(owner, acts)
	}
	for _, side := range boundaryActSides {
		matched := []Act{}
		for _, act := range acts {
			if act.Side == side {
				matched = append(matched, act)
			}
		}
		if len(matched) == 0 {
			return fmt.Errorf("%s %s side names no acts", owner, side)
		}
		if err := validateActTexts(owner+" "+side+" side", matched); err != nil {
			return err
		}
	}
	return nil
}

func validateActTexts(owner string, acts []Act) error {
	seen := map[string]bool{}
	for _, act := range acts {
		if seen[act.Text] {
			return fmt.Errorf("%s repeats act %q", owner, act.Text)
		}
		seen[act.Text] = true
	}
	return nil
}
