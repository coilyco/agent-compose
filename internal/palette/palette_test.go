package palette

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/person"
)

func TestBuildProjectsCanonicalPersonSource(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Personalities) != len(p.Personalities) {
		t.Fatalf("personalities = %d, want %d", len(doc.Personalities), len(p.Personalities))
	}
	if !reflect.DeepEqual(doc.Expressions, person.ExpressionVocabulary()) {
		t.Fatal("palette expression vocabulary drifted from the person contract")
	}
	for _, got := range doc.Personalities {
		want := p.Personalities[got.Name]
		if got.Color != want.Color || got.Motif != want.Motif ||
			got.Geometry != want.Geometry ||
			!reflect.DeepEqual(got.Emblem, want.Emblem) ||
			!reflect.DeepEqual(got.Body, want.Body) {
			t.Fatalf("palette personality %q drifted from the person contract", got.Name)
		}
	}
	if len(doc.Roles) != len(p.RoleOrder) {
		t.Fatalf("roles = %d, want %d", len(doc.Roles), len(p.RoleOrder))
	}
	for index, got := range doc.Roles {
		name := p.RoleOrder[index]
		want := p.Roles[name]
		if got.Name != name || !reflect.DeepEqual(got.Personalities, want.Personalities) {
			t.Fatalf("role[%d] = %#v, want %q with %q", index, got, name, want.Personalities)
		}
		boundary := want.FavoriteColor
		if got.Color != boundary {
			t.Fatalf("role %q color = %q, want %q", name, got.Color, boundary)
		}
	}
}

func TestMarshalAndWriteProduceValidJSON(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public", "palette.json")
	if err := Write(path, raw); err != nil {
		t.Fatal(err)
	}
}
