// Package repositoryplan owns the strict compiled repository-availability
// contract. Agent Compose is the only roles.kdl reader. Native launchers, AOS,
// Ward, and fleet convergence consume this human-readable projection.
package repositoryplan

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Format     = "agent-compose.repositories.v2"
	PolicyPath = ".agents/roles.kdl"
)

type PolicyInput struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

type Input struct {
	Identity string      `yaml:"identity"`
	Revision string      `yaml:"revision"`
	Policy   PolicyInput `yaml:"policy"`
}

type Selection struct {
	Identity   string   `yaml:"identity"`
	Path       string   `yaml:"path"`
	Source     string   `yaml:"source"`
	Scope      string   `yaml:"scope"`
	Reason     string   `yaml:"reason"`
	Required   bool     `yaml:"required,omitempty"`
	Skills     []string `yaml:"skills,omitempty"`
	Name       string   `yaml:"name,omitempty"`
	DeclaredBy string   `yaml:"declared_by,omitempty"`
	// Narrows within Skills for the owning role only, so it never rides along
	// onto a role-union residency entry.
	BindingSkills []string `yaml:"binding_skills,omitempty"`
}

type Plan struct {
	Format       string                 `yaml:"format"`
	ProjectsRoot string                 `yaml:"projects_root"`
	Inputs       []Input                `yaml:"inputs"`
	Roles        map[string][]Selection `yaml:"roles"`
	Residency    []Selection            `yaml:"residency"`
}

func Load(filename string) (Plan, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return Plan{}, fmt.Errorf("read repository plan %s: %w", filename, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return Plan{}, fmt.Errorf("parse repository plan %s: %w", filename, err)
	}
	if err := validatePlanNode(&document); err != nil {
		return Plan{}, fmt.Errorf("parse repository plan %s: %w", filename, err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Plan{}, fmt.Errorf("parse repository plan %s: trailing YAML document", filename)
	}
	var plan Plan
	if err := document.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("parse repository plan %s: %w", filename, err)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, fmt.Errorf("repository plan %s: %w", filename, err)
	}
	return plan, nil
}

func Marshal(plan Plan) ([]byte, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	document := yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			mapNode(
				"format", stringNode(plan.Format),
				"projects_root", stringNode(plan.ProjectsRoot),
				"inputs", inputsNode(plan.Inputs),
				"roles", rolesNode(plan.Roles),
				"residency", selectionsNode(plan.Residency),
			),
		},
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (p Plan) Validate() error {
	if p.Format != Format {
		return fmt.Errorf("format is %q, want %q", p.Format, Format)
	}
	root := strings.TrimSpace(p.ProjectsRoot)
	if root == "" || !filepath.IsAbs(root) {
		return fmt.Errorf("projects_root must be an absolute path")
	}
	if filepath.Clean(root) != root {
		return fmt.Errorf("projects_root must be a clean absolute path")
	}
	if len(p.Inputs) == 0 {
		return fmt.Errorf("repository plan needs at least one policy input")
	}
	inputs := map[string]bool{}
	priorInput := ""
	for index, input := range p.Inputs {
		if !validIdentity(input.Identity) {
			return fmt.Errorf("input %d has invalid identity %q", index, input.Identity)
		}
		if input.Identity <= priorInput {
			return fmt.Errorf("input identities must be strictly sorted and deduplicated")
		}
		priorInput = input.Identity
		if !validFullRevision(input.Revision) {
			return fmt.Errorf("input %q revision must be a full Git object id", input.Identity)
		}
		if input.Policy.Path != PolicyPath {
			return fmt.Errorf("input %q policy path is %q, want %q", input.Identity, input.Policy.Path, PolicyPath)
		}
		if !validDigest(input.Policy.SHA256) {
			return fmt.Errorf("input %q policy sha256 must be sha256:<64 hex chars>", input.Identity)
		}
		inputs[input.Identity] = true
	}
	if len(p.Roles) == 0 {
		return fmt.Errorf("repository plan must define roles")
	}
	for role, selections := range p.Roles {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("roles contains an empty role")
		}
		if err := validateSelections(root, inputs, "role "+fmt.Sprintf("%q", role), selections); err != nil {
			return err
		}
	}
	return validateSelections(root, inputs, "residency", p.Residency)
}

func validateSelections(root string, inputs map[string]bool, owner string, selections []Selection) error {
	prior := ""
	for index, selection := range selections {
		if !validIdentity(selection.Identity) {
			return fmt.Errorf("%s entry %d has invalid identity %q", owner, index, selection.Identity)
		}
		if selection.Identity <= prior {
			return fmt.Errorf("%s repository identities must be strictly sorted and deduplicated", owner)
		}
		prior = selection.Identity
		if !validIdentity(selection.Source) || !inputs[selection.Source] {
			return fmt.Errorf("%s repository %q source %q is not a policy input", owner, selection.Identity, selection.Source)
		}
		if !validScope(selection.Scope) || strings.TrimSpace(selection.Reason) == "" {
			return fmt.Errorf("%s repository %q needs source, scope, and reason provenance", owner, selection.Identity)
		}
		if !filepath.IsAbs(selection.Path) {
			return fmt.Errorf("%s repository %q path must be absolute", owner, selection.Identity)
		}
		cleanPath := filepath.Clean(selection.Path)
		if cleanPath != selection.Path {
			return fmt.Errorf("%s repository %q path must be clean", owner, selection.Identity)
		}
		rel, err := filepath.Rel(root, cleanPath)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%s repository %q path %s is outside projects_root %s", owner, selection.Identity, selection.Path, root)
		}
		if (selection.Name == "") != (selection.DeclaredBy == "") {
			return fmt.Errorf("%s repository %q needs both name and declared_by provider provenance", owner, selection.Identity)
		}
		if selection.Scope == "provider" && selection.Name == "" {
			return fmt.Errorf("%s repository %q provider selection needs name and declared_by provenance", owner, selection.Identity)
		}
		if selection.DeclaredBy != "" && (!validIdentity(selection.DeclaredBy) || !inputs[selection.DeclaredBy]) {
			return fmt.Errorf("%s repository %q declared_by %q is not a policy input", owner, selection.Identity, selection.DeclaredBy)
		}
		for _, skill := range selection.Skills {
			if strings.TrimSpace(skill) == "" {
				return fmt.Errorf("%s repository %q has an empty skill selector", owner, selection.Identity)
			}
		}
	}
	return nil
}

func validIdentity(value string) bool {
	if value == "" || strings.Contains(value, `\`) || path.IsAbs(value) || path.Clean(value) != value {
		return false
	}
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func validScope(value string) bool {
	switch value {
	case "operating-context", "global", "role", "provider", "role-union",
		"resident-only", "org":
		return true
	default:
		return false
	}
}

func validFullRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validHexSuffix(value, len("sha256:"), 64)
}

func validHexSuffix(value string, start, length int) bool {
	if len(value) != start+length {
		return false
	}
	_, err := hex.DecodeString(value[start:])
	return err == nil
}

func (p Plan) ForRole(role string) ([]Selection, error) {
	selections, ok := p.Roles[strings.TrimSpace(role)]
	if !ok {
		roles := make([]string, 0, len(p.Roles))
		for candidate := range p.Roles {
			roles = append(roles, candidate)
		}
		sort.Strings(roles)
		return nil, fmt.Errorf("repository plan has no role %q, available roles: %s", role, strings.Join(roles, ", "))
	}
	return append([]Selection(nil), selections...), nil
}

func validatePlanNode(document *yaml.Node) error {
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return fmt.Errorf("repository plan must be one YAML document")
	}
	root := document.Content[0]
	fields, _, err := mappingEntries(root, "repository plan")
	if err != nil {
		return err
	}
	if err := requireOnly(fields, "repository plan", "format", "projects_root", "inputs", "roles", "residency"); err != nil {
		return err
	}
	for _, field := range []string{"format", "projects_root", "inputs", "roles", "residency"} {
		if _, exists := fields[field]; !exists {
			return fmt.Errorf("repository plan missing %q", field)
		}
	}
	if err := requireString(fields["format"], "format"); err != nil {
		return err
	}
	if err := requireString(fields["projects_root"], "projects_root"); err != nil {
		return err
	}
	if err := validateInputsNode(fields["inputs"]); err != nil {
		return err
	}
	if err := validateRolesNode(fields["roles"]); err != nil {
		return err
	}
	return validateSelectionsNode(fields["residency"], "residency")
}

func validateInputsNode(node *yaml.Node) error {
	items, err := sequenceItems(node, "inputs")
	if err != nil {
		return err
	}
	for index, item := range items {
		owner := fmt.Sprintf("inputs entry %d", index)
		fields, _, err := mappingEntries(item, owner)
		if err != nil {
			return err
		}
		if err := requireOnly(fields, owner, "identity", "revision", "policy"); err != nil {
			return err
		}
		for _, field := range []string{"identity", "revision", "policy"} {
			if _, exists := fields[field]; !exists {
				return fmt.Errorf("%s missing %q", owner, field)
			}
		}
		if err := requireString(fields["identity"], owner+" identity"); err != nil {
			return err
		}
		if err := requireString(fields["revision"], owner+" revision"); err != nil {
			return err
		}
		policy, _, err := mappingEntries(fields["policy"], owner+" policy")
		if err != nil {
			return err
		}
		if err := requireOnly(policy, owner+" policy", "path", "sha256"); err != nil {
			return err
		}
		for _, field := range []string{"path", "sha256"} {
			if _, exists := policy[field]; !exists {
				return fmt.Errorf("%s policy missing %q", owner, field)
			}
			if err := requireString(policy[field], owner+" policy "+field); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRolesNode(node *yaml.Node) error {
	roles, order, err := mappingEntries(node, "roles")
	if err != nil {
		return err
	}
	prior := ""
	for _, role := range order {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("roles contains an empty role")
		}
		if role <= prior {
			return fmt.Errorf("roles must be strictly sorted and deduplicated")
		}
		prior = role
		if err := validateSelectionsNode(roles[role], "role "+fmt.Sprintf("%q", role)); err != nil {
			return err
		}
	}
	return nil
}

func validateSelectionsNode(node *yaml.Node, owner string) error {
	items, err := sequenceItems(node, owner)
	if err != nil {
		return err
	}
	for index, item := range items {
		entry := fmt.Sprintf("%s entry %d", owner, index)
		fields, _, err := mappingEntries(item, entry)
		if err != nil {
			return err
		}
		if err := requireOnly(
			fields,
			entry,
			"identity",
			"path",
			"source",
			"scope",
			"reason",
			"required",
			"skills",
			"binding_skills",
			"name",
			"declared_by",
		); err != nil {
			return err
		}
		for _, field := range []string{"identity", "path", "source", "scope", "reason"} {
			value, exists := fields[field]
			if !exists {
				return fmt.Errorf("%s missing %q", entry, field)
			}
			if err := requireString(value, entry+" "+field); err != nil {
				return err
			}
		}
		if value, exists := fields["required"]; exists {
			if err := requireBool(value, entry+" required"); err != nil {
				return err
			}
		}
		for _, field := range []string{"skills", "binding_skills"} {
			value, exists := fields[field]
			if !exists {
				continue
			}
			skills, err := sequenceItems(value, entry+" "+field)
			if err != nil {
				return err
			}
			for skillIndex, skill := range skills {
				if err := requireString(skill, fmt.Sprintf("%s %s entry %d", entry, field, skillIndex)); err != nil {
					return err
				}
			}
		}
		for _, field := range []string{"name", "declared_by"} {
			if value, exists := fields[field]; exists {
				if err := requireString(value, entry+" "+field); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mappingEntries(node *yaml.Node, owner string) (map[string]*yaml.Node, []string, error) {
	if err := requireNode(node, owner, yaml.MappingNode, "!!map"); err != nil {
		return nil, nil, err
	}
	entries := map[string]*yaml.Node{}
	order := make([]string, 0, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		if err := requireString(key, fmt.Sprintf("%s key %d", owner, index/2)); err != nil {
			return nil, nil, err
		}
		if _, exists := entries[key.Value]; exists {
			return nil, nil, fmt.Errorf("%s has duplicate key %q", owner, key.Value)
		}
		value := node.Content[index+1]
		if value.Anchor != "" || value.Kind == yaml.AliasNode || strings.HasPrefix(value.Tag, "!") && !strings.HasPrefix(value.Tag, "!!") {
			return nil, nil, fmt.Errorf("%s %q uses an unsafe YAML feature", owner, key.Value)
		}
		entries[key.Value] = value
		order = append(order, key.Value)
	}
	return entries, order, nil
}

func sequenceItems(node *yaml.Node, owner string) ([]*yaml.Node, error) {
	if err := requireNode(node, owner, yaml.SequenceNode, "!!seq"); err != nil {
		return nil, err
	}
	for index, item := range node.Content {
		if item.Anchor != "" || item.Kind == yaml.AliasNode || strings.HasPrefix(item.Tag, "!") && !strings.HasPrefix(item.Tag, "!!") {
			return nil, fmt.Errorf("%s entry %d uses an unsafe YAML feature", owner, index)
		}
	}
	return node.Content, nil
}

func requireString(node *yaml.Node, owner string) error {
	return requireNode(node, owner, yaml.ScalarNode, "!!str")
}

func requireBool(node *yaml.Node, owner string) error {
	if err := requireNode(node, owner, yaml.ScalarNode, "!!bool"); err != nil {
		return err
	}
	if node.Value != "true" && node.Value != "false" {
		return fmt.Errorf("%s must be a boolean scalar", owner)
	}
	return nil
}

func requireNode(node *yaml.Node, owner string, kind yaml.Kind, tag string) error {
	if node == nil {
		return fmt.Errorf("%s is missing", owner)
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return fmt.Errorf("%s uses an unsafe YAML feature", owner)
	}
	if node.Kind != kind || node.Tag != tag {
		return fmt.Errorf("%s must be a %s node", owner, strings.TrimPrefix(tag, "!!"))
	}
	return nil
}

func requireOnly(fields map[string]*yaml.Node, owner string, allowed ...string) error {
	set := map[string]bool{}
	for _, field := range allowed {
		set[field] = true
	}
	for field := range fields {
		if !set[field] {
			return fmt.Errorf("%s has unknown field %q", owner, field)
		}
	}
	return nil
}

func inputsNode(inputs []Input) *yaml.Node {
	items := make([]*yaml.Node, 0, len(inputs))
	for _, input := range inputs {
		items = append(items, mapNode(
			"identity", stringNode(input.Identity),
			"revision", stringNode(input.Revision),
			"policy", mapNode(
				"path", stringNode(input.Policy.Path),
				"sha256", stringNode(input.Policy.SHA256),
			),
		))
	}
	return sequenceNode(items...)
}

func rolesNode(roles map[string][]Selection) *yaml.Node {
	keys := make([]string, 0, len(roles))
	for role := range roles {
		keys = append(keys, role)
	}
	sort.Strings(keys)
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, role := range keys {
		node.Content = append(node.Content, stringNode(role), selectionsNode(roles[role]))
	}
	return node
}

func selectionsNode(selections []Selection) *yaml.Node {
	items := make([]*yaml.Node, 0, len(selections))
	for _, selection := range selections {
		fields := []*yaml.Node{
			stringNode("identity"), stringNode(selection.Identity),
			stringNode("path"), stringNode(selection.Path),
			stringNode("source"), stringNode(selection.Source),
			stringNode("scope"), stringNode(selection.Scope),
			stringNode("reason"), stringNode(selection.Reason),
		}
		if selection.Required {
			fields = append(fields, stringNode("required"), boolNode(selection.Required))
		}
		for _, selector := range []struct {
			key      string
			patterns []string
		}{
			{"skills", selection.Skills},
			{"binding_skills", selection.BindingSkills},
		} {
			if len(selector.patterns) == 0 {
				continue
			}
			patterns := make([]*yaml.Node, 0, len(selector.patterns))
			for _, pattern := range selector.patterns {
				patterns = append(patterns, stringNode(pattern))
			}
			fields = append(fields, stringNode(selector.key), sequenceNode(patterns...))
		}
		if selection.Name != "" {
			fields = append(fields, stringNode("name"), stringNode(selection.Name))
		}
		if selection.DeclaredBy != "" {
			fields = append(fields, stringNode("declared_by"), stringNode(selection.DeclaredBy))
		}
		items = append(items, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: fields})
	}
	return sequenceNode(items...)
}

func mapNode(pairs ...any) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for index := 0; index < len(pairs); index += 2 {
		key := pairs[index].(string)
		value := pairs[index+1].(*yaml.Node)
		node.Content = append(node.Content, stringNode(key), value)
	}
	return node
}

func sequenceNode(items ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: items}
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func boolNode(value bool) *yaml.Node {
	if value {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
}
