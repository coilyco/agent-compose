package person

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSnapshotRoundTripsCompletePersonModel(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	first, err := MarshalSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("person snapshot must be deterministic")
	}

	var snapshot Snapshot
	if err := json.Unmarshal(first, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Format != SnapshotFormat ||
		snapshot.SchemaVersion != SnapshotSchemaVersion ||
		snapshot.Source != p.ProviderID() ||
		snapshot.Person != p.Name {
		t.Fatalf("unexpected snapshot identity: %+v", snapshot)
	}
	if !reflect.DeepEqual(snapshot.RoleOrder, p.RoleOrder) ||
		len(snapshot.Roles) != len(p.Roles) {
		t.Fatalf("snapshot role inventory drifted: %+v", snapshot.RoleOrder)
	}
	for _, name := range p.RoleOrder {
		got, ok := snapshot.Roles[name]
		if !ok {
			t.Fatalf("snapshot omitted role %q", name)
		}
		want := p.Roles[name]
		if !reflect.DeepEqual(got.Role, want) {
			t.Fatalf("snapshot role %q = %+v, want %+v", name, got.Role, want)
		}
		favorite := want.FavoriteColor
		if got.FavoriteColor != favorite {
			t.Fatalf("snapshot role %q favorite = %q, want %q", name, got.FavoriteColor, favorite)
		}
	}
	if !reflect.DeepEqual(snapshot.Personalities, p.Personalities) {
		t.Fatal("snapshot personality catalog drifted from the loaded person")
	}
	if !reflect.DeepEqual(snapshot.Expressions, ExpressionVocabulary()) {
		t.Fatal("snapshot expression vocabulary drifted from the person contract")
	}
}

func TestSnapshotHasAnExplicitPersonFieldPolicy(t *testing.T) {
	covered := map[string]bool{
		"ProviderKind":         true,
		"Name":                 true,
		"Roles":                true,
		"RoleOrder":            true,
		"Boundaries":           true,
		"BoundaryOrder":        true,
		"boundarySkills":       true,
		"Personalities":        true,
		"PersonalityOrder":     true,
		"Guardrails":           true,
		"GuardrailOrder":       true,
		"guardrailSkills":      true,
		"Raw":                  true,
		"Libraries":            true,
		"PersonalityLibraries": true,
		"roleSkills":           true,
		"roleMethods":          true,
		"source":               true,
	}
	model := reflect.TypeOf(Person{})
	for index := range model.NumField() {
		field := model.Field(index)
		if !covered[field.Name] {
			t.Fatalf("person field %q needs an explicit snapshot export decision", field.Name)
		}
	}
	if len(covered) != model.NumField() {
		t.Fatalf("snapshot field policy names %d fields, person model has %d", len(covered), model.NumField())
	}
}

func TestSnapshotV4RejectsInconsistentProvenanceAndAffinities(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := BuildSnapshotV4(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSnapshotV4(snapshot); err != nil {
		t.Fatal(err)
	}
	entry := snapshot.Personalities["tenacious"]
	entry.Digest = "bad"
	snapshot.Personalities["tenacious"] = entry
	if err := ValidateSnapshotV4(snapshot); err == nil {
		t.Fatal("invalid personality provenance passed")
	}

	snapshot, err = BuildSnapshotV4(p)
	if err != nil {
		t.Fatal(err)
	}
	entry = snapshot.Personalities["tenacious"]
	entry.Affinities[0].Personalities = []string{"tenacious"}
	snapshot.Personalities["tenacious"] = entry
	if err := ValidateSnapshotV4(snapshot); err == nil {
		t.Fatal("inconsistent affinity boundary passed")
	}
}
