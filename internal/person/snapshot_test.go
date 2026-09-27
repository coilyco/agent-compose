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

// withheld marks a person field the v3 snapshot deliberately does not carry.
const withheld = ""

// snapshotExport maps each person field to the Snapshot field that carries it,
// or to withheld. See docs/person-packages.md.
var snapshotExport = map[string]string{
	"ProviderKind":         "Source",
	"Name":                 "Person",
	"Roles":                "Roles",
	"RoleOrder":            "RoleOrder",
	"Boundaries":           "Boundaries",
	"BoundaryOrder":        "BoundaryOrder",
	"Personalities":        "Personalities",
	"Guardrails":           "Guardrails",
	"GuardrailOrder":       "GuardrailOrder",
	"PersonalityOrder":     withheld,
	"boundarySkills":       withheld,
	"guardrailSkills":      withheld,
	"Raw":                  withheld,
	"Libraries":            withheld,
	"PersonalityLibraries": withheld,
	"roleSkills":           withheld,
	"roleMethods":          withheld,
	"source":               withheld,
}

func TestSnapshotHasAnExplicitPersonFieldPolicy(t *testing.T) {
	model := reflect.TypeOf(Person{})
	for index := range model.NumField() {
		field := model.Field(index)
		if _, ok := snapshotExport[field.Name]; !ok {
			t.Fatalf("person field %q needs an explicit snapshot export decision", field.Name)
		}
	}
	if len(snapshotExport) != model.NumField() {
		t.Fatalf("snapshot field policy names %d fields, person model has %d", len(snapshotExport), model.NumField())
	}
}

// The policy above recorded only that someone had looked at each field, so both
// a dropped export and a quietly added one stayed green. This checks both ways.
func TestSnapshotExportPolicyMatchesTheSnapshotType(t *testing.T) {
	snapshot := reflect.TypeOf(Snapshot{})
	for name, target := range snapshotExport {
		if target == withheld {
			if _, exported := snapshot.FieldByName(name); exported {
				t.Fatalf("person field %q is withheld but Snapshot carries it; update the policy", name)
			}
			continue
		}
		if _, ok := snapshot.FieldByName(target); !ok {
			t.Fatalf("person field %q claims Snapshot field %q, which does not exist", name, target)
		}
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

func TestSnapshotLetsOnlyADerivedRoleKeepItsParentsGuardrail(t *testing.T) {
	for _, test := range []struct {
		role    string
		derives string
		wantErr bool
	}{
		{role: "researcher", derives: "scientist"},
		{role: "game-design", derives: "game-dev", wantErr: true},
		{role: "eng-platform", wantErr: true},
	} {
		p, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		role := p.Roles[test.role]
		role.Derives = test.derives
		role.Guardrail = "provable-results"
		p.Roles[test.role] = role
		_, err = BuildSnapshot(p)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Errorf("role %q deriving %q with scientist's guardrail: err = %v, want error %v",
				test.role, test.derives, err, test.wantErr)
		}
	}
}
