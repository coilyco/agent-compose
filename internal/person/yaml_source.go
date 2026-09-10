package person

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// A package authors every section fragment in one format. Mixing them is
// refused by name rather than half-read. See docs/person-packages.md.

type sectionFile struct {
	directory string
	node      string
	name      string
	slug      string
	raw       []byte
}

// readSectionFiles collects every fragment the package declares, in section
// then filename order, which is what fixes each entity's order.
func readSectionFiles(source fs.FS, label string, sections []struct {
	directory string
	node      string
},
) ([]sectionFile, error) {
	var files []sectionFile
	for _, section := range sections {
		entries, err := fs.ReadDir(source, section.directory)
		if os.IsNotExist(err) && section.directory != "roles" {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: read person %s: %w", label, section.directory, err)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("%s: person %s section is empty", label, section.directory)
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				if section.directory == "roles" {
					if _, ok := pRoleSkillDirectory(entry.Name()); ok {
						continue
					}
				}
				return nil, fmt.Errorf(
					"%s: person %s has unexpected entry %q", label, section.directory, entry.Name())
			}
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			slug, ok := personFragmentSlug(name)
			if !ok {
				return nil, fmt.Errorf(
					"%s: person %s has unexpected entry %q", label, section.directory, name)
			}
			path := section.directory + "/" + name
			raw, err := fs.ReadFile(source, path)
			if err != nil {
				return nil, fmt.Errorf("%s: read person fragment %q: %w", label, path, err)
			}
			files = append(files, sectionFile{
				directory: section.directory,
				node:      section.node,
				name:      path,
				slug:      slug,
				raw:       raw,
			})
		}
	}
	return files, nil
}

// yamlSectionFiles reports whether the package is authored as YAML, refusing a
// package that mixes the two spellings.
func yamlSectionFiles(label string, files []sectionFile) (bool, error) {
	var yamlNames, kdlNames []string
	for _, file := range files {
		if strings.HasSuffix(file.name, yamlFragmentExt) {
			yamlNames = append(yamlNames, file.name)
			continue
		}
		kdlNames = append(kdlNames, file.name)
	}
	if len(yamlNames) > 0 && len(kdlNames) > 0 {
		return false, fmt.Errorf(
			"%s: person mixes YAML and KDL fragments; YAML %s, KDL %s",
			label, strings.Join(yamlNames, ", "), strings.Join(kdlNames, ", "))
	}
	return len(yamlNames) > 0, nil
}

// buildYAMLPerson decodes every fragment into the model directly, so the
// authored file and the loaded entity share one shape.
func buildYAMLPerson(kind, name string, files []sectionFile, label string) (*Person, error) {
	p := &Person{
		ProviderKind:   kind,
		Name:           name,
		Roles:          map[string]Role{},
		Boundaries:     map[string]Boundary{},
		Personalities:  map[string]Personality{},
		roleSkills:     map[string][]byte{},
		roleMethods:    map[string]map[string][]byte{},
		boundarySkills: map[string][]byte{},
	}
	var raw bytes.Buffer
	for _, file := range files {
		raw.WriteString(file.name)
		raw.WriteByte(0)
		raw.Write(file.raw)
		raw.WriteByte('\n')

		switch file.node {
		case "role":
			var entity yamlRoleEntity
			if err := decodeEntity(file.raw, &entity); err != nil {
				return nil, fmt.Errorf("%s: person fragment %q: %w", label, file.name, err)
			}
			if entity.Role != file.slug {
				return nil, fmt.Errorf(
					"%s: person fragment %q filename does not match role %q",
					label, file.name, entity.Role)
			}
			if _, dup := p.Roles[entity.Role]; dup {
				return nil, fmt.Errorf("%s: person role %q declared twice", label, entity.Role)
			}
			p.Roles[entity.Role] = entity.model()
			p.RoleOrder = append(p.RoleOrder, entity.Role)
		case "personality":
			var entity yamlPersonalityEntity
			if err := decodeEntity(file.raw, &entity); err != nil {
				return nil, fmt.Errorf("%s: person fragment %q: %w", label, file.name, err)
			}
			if entity.Personality != file.slug {
				return nil, fmt.Errorf(
					"%s: person fragment %q filename does not match personality %q",
					label, file.name, entity.Personality)
			}
			if _, dup := p.Personalities[entity.Personality]; dup {
				return nil, fmt.Errorf(
					"%s: person personality %q declared twice", label, entity.Personality)
			}
			p.Personalities[entity.Personality] = entity.model()
			p.PersonalityOrder = append(p.PersonalityOrder, entity.Personality)
		case "boundary":
			var entity yamlBoundaryEntity
			if err := decodeEntity(file.raw, &entity); err != nil {
				return nil, fmt.Errorf("%s: person fragment %q: %w", label, file.name, err)
			}
			if entity.Boundary != file.slug {
				return nil, fmt.Errorf(
					"%s: person fragment %q filename does not match boundary %q",
					label, file.name, entity.Boundary)
			}
			if _, dup := p.Boundaries[entity.Boundary]; dup {
				return nil, fmt.Errorf(
					"%s: person boundary %q declared twice", label, entity.Boundary)
			}
			p.Boundaries[entity.Boundary] = entity.model()
			p.BoundaryOrder = append(p.BoundaryOrder, entity.Boundary)
		case "guardrail":
			var entity yamlGuardrailEntity
			if err := decodeEntity(file.raw, &entity); err != nil {
				return nil, fmt.Errorf("%s: person fragment %q: %w", label, file.name, err)
			}
			if entity.Guardrail != file.slug {
				return nil, fmt.Errorf(
					"%s: person fragment %q filename does not match guardrail %q",
					label, file.name, entity.Guardrail)
			}
			if _, dup := p.Guardrails[entity.Guardrail]; dup {
				return nil, fmt.Errorf(
					"%s: person guardrail %q declared twice", label, entity.Guardrail)
			}
			if p.Guardrails == nil {
				p.Guardrails = map[string]Guardrail{}
			}
			p.Guardrails[entity.Guardrail] = entity.model()
			p.GuardrailOrder = append(p.GuardrailOrder, entity.Guardrail)
		}
	}
	p.Raw = raw.Bytes()
	if err := validateDecodedPerson(p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return p, nil
}
