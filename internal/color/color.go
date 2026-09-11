// Package color owns favorite-color math: hex parsing, the terminal-legible
// OKLab band, and derived favorites for multi-component identities.
package color

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// The legible band keeps colors readable on dark and light terminals:
// mid lightness, never gray.
const (
	minL      = 0.60
	maxL      = 0.80
	minChroma = 0.05
)

// Shimmer highlights sit above the legible band on purpose: they are a pulse
// against the base color, never a color text is read against.
const (
	shimmerLift    = 0.10
	shimmerCeiling = 0.95
)

type okLab struct {
	L, A, B float64
}

func (c okLab) chroma() float64 {
	return math.Hypot(c.A, c.B)
}

// Legible validates a hex color against the band; it is the parse-time
// gate for personality colors.
func Legible(hex string) error {
	lab, err := toOKLab(hex)
	if err != nil {
		return err
	}
	if lab.L < minL || lab.L > maxL {
		return fmt.Errorf("color %s has OKLab lightness %.2f outside the terminal-legible band [%.2f, %.2f]", hex, lab.L, minL, maxL)
	}
	if lab.chroma() < minChroma {
		return fmt.Errorf("color %s is too gray (chroma %.3f, floor %.3f)", hex, lab.chroma(), minChroma)
	}
	return nil
}

// Favorite derives one color from component colors: the OKLab centroid with
// chroma restored to the components' minimum, clamped into the band.
func Favorite(hexes []string) (string, error) {
	if len(hexes) == 0 {
		return "", fmt.Errorf("favorite needs at least one component color")
	}
	var sum okLab
	minComponentChroma := math.Inf(1)
	for _, hex := range hexes {
		lab, err := toOKLab(hex)
		if err != nil {
			return "", err
		}
		sum.L += lab.L
		sum.A += lab.A
		sum.B += lab.B
		minComponentChroma = math.Min(minComponentChroma, lab.chroma())
	}
	n := float64(len(hexes))
	blend := okLab{L: sum.L / n, A: sum.A / n, B: sum.B / n}
	if c := blend.chroma(); c > 0 && c < minComponentChroma {
		scale := minComponentChroma / c
		blend.A *= scale
		blend.B *= scale
	}
	blend.L = math.Max(minL, math.Min(maxL, blend.L))
	if blend.chroma() < minChroma {
		return "", fmt.Errorf("derived favorite fell below the gray floor; give the components more chroma")
	}
	return fromOKLab(blend), nil
}

// driftCap limits how far Favorites may move a group's color from its own
// blend. shareWeight sets how sharply a shared component is discounted.
const (
	driftCap     = 0.03
	spreadRounds = 400
	spreadStep   = 0.004
	shareWeight  = 2
)

// Favorites derives every group's color together, weighting each component by
// how few groups share it, then spreading the results. See RenderSnapshot.
func Favorites(groups [][]string) ([]string, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("favorites needs at least one group")
	}
	shared := map[string]int{}
	for _, group := range groups {
		seen := map[string]bool{}
		for _, hex := range group {
			if !seen[hex] {
				shared[hex]++
				seen[hex] = true
			}
		}
	}
	anchors := make([]okLab, 0, len(groups))
	for index, group := range groups {
		weights := make([]float64, 0, len(group))
		for _, hex := range group {
			weights = append(weights, 1/math.Pow(float64(shared[hex]), shareWeight))
		}
		blended, err := weightedBlend(group, weights)
		if err != nil {
			return nil, fmt.Errorf("group %d: %w", index, err)
		}
		anchors = append(anchors, blended)
	}
	spread := spreadApart(anchors)
	out := make([]string, 0, len(spread))
	for index, lab := range spread {
		hex := fromOKLab(lab)
		if err := Legible(hex); err != nil {
			return nil, fmt.Errorf("group %d: %w", index, err)
		}
		out = append(out, hex)
	}
	return out, nil
}

// MinSeparation reports the smallest OKLab distance between any two of the
// colors, so a caller can assert a palette stays tellable apart.
func MinSeparation(hexes []string) (float64, error) {
	if len(hexes) < 2 {
		return math.Inf(1), nil
	}
	labs := make([]okLab, 0, len(hexes))
	for _, hex := range hexes {
		lab, err := toOKLab(hex)
		if err != nil {
			return 0, err
		}
		labs = append(labs, lab)
	}
	return minSeparation(labs), nil
}

func weightedBlend(hexes []string, weights []float64) (okLab, error) {
	var sum okLab
	total := 0.0
	minComponentChroma := math.Inf(1)
	for index, hex := range hexes {
		lab, err := toOKLab(hex)
		if err != nil {
			return okLab{}, err
		}
		w := weights[index]
		sum.L += w * lab.L
		sum.A += w * lab.A
		sum.B += w * lab.B
		total += w
		minComponentChroma = math.Min(minComponentChroma, lab.chroma())
	}
	blend := okLab{L: sum.L / total, A: sum.A / total, B: sum.B / total}
	if c := blend.chroma(); c > 0 && c < minComponentChroma {
		scale := minComponentChroma / c
		blend.A *= scale
		blend.B *= scale
	}
	return settle(blend), nil
}

// spreadApart walks every group away from its nearest neighbor, projecting
// back inside its drift ball each time, and keeps the best round it saw.
func spreadApart(anchors []okLab) []okLab {
	current := append([]okLab(nil), anchors...)
	best := append([]okLab(nil), anchors...)
	bestSeparation := minSeparation(current)
	for round := 0; round < spreadRounds; round++ {
		for index := range current {
			nearest, distance := -1, math.Inf(1)
			for other := range current {
				if other == index {
					continue
				}
				if d := separation(current[index], current[other]); d < distance {
					nearest, distance = other, d
				}
			}
			if nearest < 0 {
				continue
			}
			current[index] = project(step(current[index], current[nearest]), anchors[index])
		}
		if s := minSeparation(current); s > bestSeparation {
			bestSeparation = s
			best = append([]okLab(nil), current...)
		}
	}
	return best
}

func step(from, away okLab) okLab {
	dL, dA, dB := from.L-away.L, from.A-away.A, from.B-away.B
	norm := math.Sqrt(dL*dL + dA*dA + dB*dB)
	if norm < 1e-9 {
		dL, dA, dB, norm = 1, 0, 0, 1
	}
	return okLab{
		L: from.L + dL*spreadStep/norm,
		A: from.A + dA*spreadStep/norm,
		B: from.B + dB*spreadStep/norm,
	}
}

func project(candidate, anchor okLab) okLab {
	dL, dA, dB := candidate.L-anchor.L, candidate.A-anchor.A, candidate.B-anchor.B
	if drift := math.Sqrt(dL*dL + dA*dA + dB*dB); drift > driftCap {
		scale := driftCap / drift
		candidate = okLab{
			L: anchor.L + dL*scale,
			A: anchor.A + dA*scale,
			B: anchor.B + dB*scale,
		}
	}
	return settle(candidate)
}

// settle clamps lightness into the band, pulls chroma down until the color is
// representable, then lifts it back over the gray floor.
func settle(lab okLab) okLab {
	lab.L = math.Max(minL, math.Min(maxL, lab.L))
	for attempt := 0; attempt < 64 && !inGamut(lab); attempt++ {
		lab.A *= 0.98
		lab.B *= 0.98
	}
	if c := lab.chroma(); c > 0 && c < minChroma {
		scale := minChroma / c
		lab.A *= scale
		lab.B *= scale
	}
	return lab
}

func separation(a, b okLab) float64 {
	return math.Sqrt(sqf(a.L-b.L) + sqf(a.A-b.A) + sqf(a.B-b.B))
}

func minSeparation(labs []okLab) float64 {
	lowest := math.Inf(1)
	for i := 0; i < len(labs); i++ {
		for j := i + 1; j < len(labs); j++ {
			lowest = math.Min(lowest, separation(labs[i], labs[j]))
		}
	}
	return lowest
}

func inGamut(lab okLab) bool {
	l := cube(lab.L + 0.3963377774*lab.A + 0.2158037573*lab.B)
	m := cube(lab.L - 0.1055613458*lab.A - 0.0638541728*lab.B)
	s := cube(lab.L - 0.0894841775*lab.A - 1.2914855480*lab.B)
	for _, channel := range []float64{
		4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
	} {
		if channel < -0.0001 || channel > 1.0001 {
			return false
		}
	}
	return true
}

func sqf(v float64) float64 {
	return v * v
}

// Shimmer lightens a color for the paired highlight token a terminal theme
// pulses against the base. Lightness is raised, hue and chroma are untouched.
func Shimmer(hex string) (string, error) {
	lab, err := toOKLab(hex)
	if err != nil {
		return "", err
	}
	lab.L = math.Min(shimmerCeiling, lab.L+shimmerLift)
	return fromOKLab(lab), nil
}

// Nearest picks the candidate closest to hex in hue and chroma, ignoring
// lightness. It selects a slot from a fixed palette rather than a free color.
func Nearest(hex string, candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("nearest needs at least one candidate color")
	}
	target, err := toOKLab(hex)
	if err != nil {
		return "", err
	}
	best := ""
	bestDistance := math.Inf(1)
	for _, candidate := range candidates {
		lab, err := toOKLab(candidate)
		if err != nil {
			return "", err
		}
		distance := math.Hypot(lab.A-target.A, lab.B-target.B)
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best, nil
}

// ANSI renders text in the color: truecolor when supported, else the
// nearest 256-color index.
func ANSI(hex, text string, trueColor bool) string {
	r, g, b, err := parseHex(hex)
	if err != nil {
		return text
	}
	if trueColor {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
	}
	return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", nearest256(r, g, b), text)
}

func parseHex(hex string) (int, int, int, error) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) != 6 {
		return 0, 0, 0, fmt.Errorf("color %q must be #rrggbb", hex)
	}
	var r, g, b int
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, 0, 0, fmt.Errorf("color %q must be #rrggbb: %w", hex, err)
	}
	return r, g, b, nil
}

func toOKLab(hex string) (okLab, error) {
	ri, gi, bi, err := parseHex(hex)
	if err != nil {
		return okLab{}, err
	}
	r, g, b := linearize(ri), linearize(gi), linearize(bi)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return okLab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}, nil
}

func fromOKLab(lab okLab) string {
	l := cube(lab.L + 0.3963377774*lab.A + 0.2158037573*lab.B)
	m := cube(lab.L - 0.1055613458*lab.A - 0.0638541728*lab.B)
	s := cube(lab.L - 0.0894841775*lab.A - 1.2914855480*lab.B)
	r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	return fmt.Sprintf("#%02x%02x%02x", delinearize(r), delinearize(g), delinearize(b))
}

func linearize(channel int) float64 {
	c := float64(channel) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func delinearize(c float64) int {
	c = math.Max(0, math.Min(1, c))
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return int(math.Round(c * 255))
}

func cube(v float64) float64 {
	return v * v * v
}

func nearest256(r, g, b int) int {
	toCube := func(v int) int {
		if v < 48 {
			return 0
		}
		if v < 115 {
			return 1
		}
		return (v - 35) / 40
	}
	levels := []int{0, 95, 135, 175, 215, 255}
	cr, cg, cb := toCube(r), toCube(g), toCube(b)
	cubeIndex := 16 + 36*cr + 6*cg + cb
	cubeDist := sq(r-levels[cr]) + sq(g-levels[cg]) + sq(b-levels[cb])

	gray := (r + g + b) / 3
	grayIndex := 232 + max(0, min(23, (gray-8)/10))
	grayLevel := 8 + 10*(grayIndex-232)
	grayDist := sq(r-grayLevel) + sq(g-grayLevel) + sq(b-grayLevel)

	if grayDist < cubeDist {
		return grayIndex
	}
	return cubeIndex
}

func sq(v int) int {
	return v * v
}

// A window background sits far below the legible band and holds almost no chroma.
// Lightness buys separation as the roster grows, not chroma: role-palette.txt.
const (
	backgroundL      = 0.260
	backgroundChroma = 0.055
)

// Backgrounds solves one window background per accent against the whole set,
// which no consumer holding a single role can do.
func Backgrounds(accents []string) ([]string, error) {
	if len(accents) == 0 {
		return nil, fmt.Errorf("backgrounds needs at least one accent")
	}
	hues := make([]float64, 0, len(accents))
	for index, hex := range accents {
		lab, err := toOKLab(hex)
		if err != nil {
			return nil, fmt.Errorf("accent %d: %w", index, err)
		}
		hues = append(hues, math.Atan2(lab.B, lab.A))
	}
	for index, hue := range spreadHues(hues) {
		accents[index] = fromOKLab(okLab{
			L: backgroundL,
			A: backgroundChroma * math.Cos(hue),
			B: backgroundChroma * math.Sin(hue),
		})
	}
	return accents, nil
}

// spreadHues moves the set onto equal spacing while keeping each role's hue
// order, at the rotation offset that turns every role the least.
func spreadHues(hues []float64) []float64 {
	order := make([]int, len(hues))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(first, second int) bool {
		return hues[order[first]] < hues[order[second]]
	})
	gap := 2 * math.Pi / float64(len(hues))
	// The offset that minimises total rotation is the circular mean of each
	// role's distance from the slot it would take.
	sumSin, sumCos := 0.0, 0.0
	for slot, index := range order {
		delta := hues[index] - float64(slot)*gap
		sumSin += math.Sin(delta)
		sumCos += math.Cos(delta)
	}
	offset := math.Atan2(sumSin, sumCos)
	spread := make([]float64, len(hues))
	for slot, index := range order {
		spread[index] = offset + float64(slot)*gap
	}
	return spread
}
