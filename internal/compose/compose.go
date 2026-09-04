// Package compose orchestrates one composition end to end so the CLI stays a
// thin rendering layer over the same path the tests exercise.
package compose

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

type Result struct {
	Resolution   *resolver.Resolution
	Bundle       *bundle.Result
	ExternalOnly bool
}

type Options struct {
	PersonPolicy         string
	PersonSource         string
	PersonalityLibraries []string
	// OperatingBase is the host doctrine that leads the bundle's instructions,
	// so a projected role no longer depends on a host-owned load point.
	OperatingBase string
	// OperatingAppendix is the configured tail, kept out of OperatingBase so the
	// roster-card rewrite cannot reach it. agent-compose#6987.
	OperatingAppendix string
}

// RootSource names one trusted provider root selected by a host launcher.
type RootSource struct {
	ID            string
	Root          string
	Reason        string
	Scope         string
	Skills        []string
	BindingSkills []string
}

type externalOnlyError struct {
	err error
}

func (e *externalOnlyError) Error() string {
	return e.err.Error()
}

func (e *externalOnlyError) Unwrap() error {
	return e.err
}

// IsExternalOnlyError reports whether composition failed after resolving an
// external-only policy. Launchers use it to prohibit unsafe fallback.
func IsExternalOnlyError(err error) bool {
	var target *externalOnlyError
	return errors.As(err, &target)
}

func Run(requestPath, outDir string) (*Result, error) {
	return RunWithOptions(requestPath, outDir, Options{})
}

// RunWithOptions applies a host person selection beneath any explicit
// request-local selection.
func RunWithOptions(requestPath, outDir string, opts Options) (*Result, error) {
	req, err := schema.ParseRequest(requestPath)
	if err != nil {
		return nil, err
	}
	hostExternalOnly := opts.PersonPolicy == personpolicy.ExternalOnly
	if err := personpolicy.Validate(opts.PersonPolicy, opts.PersonSource); err != nil {
		return nil, wrapPolicyError(err, hostExternalOnly)
	}
	externalOnly := hostExternalOnly ||
		req.PersonPolicy == personpolicy.ExternalOnly
	personSource := opts.PersonSource
	libraries := append([]string(nil), opts.PersonalityLibraries...)
	if req.PersonSource != "" {
		personSource = filepath.Join(filepath.Dir(requestPath), req.PersonSource)
	}
	for _, library := range req.PersonalityLibraries {
		if person.IsCoreLibrary(library) {
			libraries = append(libraries, library)
			continue
		}
		libraries = append(libraries, filepath.Join(filepath.Dir(requestPath), library))
	}
	if err := personpolicy.Validate(effectivePolicy(externalOnly), personSource); err != nil {
		return nil, wrapPolicyError(err, externalOnly)
	}

	p, err := person.Load()
	if personSource != "" {
		p, err = person.LoadDirectoryWithLibraries(personSource, libraries...)
	} else if len(libraries) > 0 {
		return nil, wrapPolicyError(fmt.Errorf("personality-library requires a selected local person-source"), externalOnly)
	}
	if err != nil {
		return nil, wrapPolicyError(err, externalOnly)
	}
	sources, missing, err := schema.LoadSources(req, requestPath)
	if err != nil {
		return nil, wrapPolicyError(err, externalOnly)
	}
	return materialize(req, p, sources, missing, outDir, externalOnly, "", "")
}

// RunRoots composes trusted absolute provider roots selected by a host
// launcher without weakening the portable KDL path rules.
func RunRoots(
	req *schema.Request,
	roots []RootSource,
	outDir string,
	opts Options,
) (*Result, error) {
	return RunRootsWithMissing(req, roots, nil, outDir, opts)
}

// RunRootsWithMissing preserves explicit provider exclusions in the resolver
// trace while composing the trusted roots that remain available.
func RunRootsWithMissing(
	req *schema.Request,
	roots []RootSource,
	missing []schema.MissingSource,
	outDir string,
	opts Options,
) (*Result, error) {
	hostExternalOnly := opts.PersonPolicy == personpolicy.ExternalOnly
	if err := personpolicy.Validate(opts.PersonPolicy, opts.PersonSource); err != nil {
		return nil, wrapPolicyError(err, hostExternalOnly)
	}
	p, err := person.Load()
	if opts.PersonSource != "" {
		p, err = person.LoadDirectoryWithLibraries(
			opts.PersonSource,
			opts.PersonalityLibraries...,
		)
	} else if len(opts.PersonalityLibraries) > 0 {
		err = fmt.Errorf("personality-library requires a selected local person-source")
	}
	if err != nil {
		return nil, wrapPolicyError(err, hostExternalOnly)
	}
	sources := make([]*schema.Source, 0, len(roots))
	for _, root := range roots {
		source, err := schema.LoadSource(root.Root)
		if err != nil {
			return nil, wrapPolicyError(
				fmt.Errorf("source %q: %w", root.ID, err),
				hostExternalOnly,
			)
		}
		source.ID = root.ID
		source.AdmissionReason = root.Reason
		source.ProviderScope = root.Scope
		if err := schema.SelectOrdinarySkills(source, root.Skills, root.BindingSkills); err != nil {
			return nil, wrapPolicyError(
				fmt.Errorf("source %q: %w", root.ID, err),
				hostExternalOnly,
			)
		}
		sources = append(sources, source)
	}
	return materialize(req, p, sources, missing, outDir, hostExternalOnly, opts.OperatingBase, opts.OperatingAppendix)
}

func materialize(
	req *schema.Request,
	p *person.Person,
	sources []*schema.Source,
	missing []schema.MissingSource,
	outDir string,
	externalOnly bool,
	operatingBase string,
	operatingAppendix string,
) (*Result, error) {
	// Applied here because both entry points funnel through materialize, and
	// everything downstream reads the identity off the person.
	if req.Identity != nil {
		if err := p.OverrideRoleIdentity(req.Role, req.Identity.Name, req.Identity.Pronouns); err != nil {
			return nil, wrapPolicyError(err, externalOnly)
		}
	}
	res, err := resolver.Resolve(req, p, sources, missing)
	if err != nil {
		return nil, wrapPolicyError(err, externalOnly)
	}
	res.OperatingBase = operatingBase
	res.OperatingAppendix = operatingAppendix
	b, err := bundle.Materialize(res, outDir)
	if err != nil {
		return nil, wrapPolicyError(err, externalOnly)
	}
	return &Result{
		Resolution:   res,
		Bundle:       b,
		ExternalOnly: externalOnly,
	}, nil
}

func effectivePolicy(externalOnly bool) string {
	if externalOnly {
		return personpolicy.ExternalOnly
	}
	return ""
}

func wrapPolicyError(err error, externalOnly bool) error {
	if externalOnly {
		return &externalOnlyError{err: err}
	}
	return err
}
