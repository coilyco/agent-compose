package person

import (
	"fmt"
)

// validateActs checks an unsided list against actsPerAttribute..maxActs. A
// maxActs of zero marks a sided boundary list, which needs exactly three per side.
func validateActs(owner string, acts []Act, maxActs int) error {
	if maxActs > 0 {
		if len(acts) < actsPerAttribute || len(acts) > maxActs {
			return fmt.Errorf(
				"%s names %d acts, needs %d to %d", owner, len(acts), actsPerAttribute, maxActs,
			)
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
		if len(matched) != actsPerAttribute {
			return fmt.Errorf(
				"%s %s side names %d acts, needs exactly %d",
				owner, side, len(matched), actsPerAttribute,
			)
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
