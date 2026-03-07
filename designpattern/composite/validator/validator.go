package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator validates a string value.
// Leaf validators (NonEmpty, MinLength, ...) check a single rule.
// Composite validators (And, Or, Not) combine children into a tree,
// mirroring io.MultiReader's composition of io.Reader.
type Validator interface {
	Validate(value string) error
}

// ---------------------------------------------------------------------------
// Composites
// ---------------------------------------------------------------------------

// And returns a composite that succeeds only when ALL children succeed.
// It short-circuits on the first error, like && in boolean expressions.
//
// This is the direct analog of io.MultiWriter: one write call fans out to
// all underlying writers; one Validate call fans out to all child validators.
func And(validators ...Validator) Validator {
	return &andValidator{validators: validators}
}

type andValidator struct {
	validators []Validator
}

func (a *andValidator) Validate(value string) error {
	for _, v := range a.validators {
		if err := v.Validate(value); err != nil {
			return err
		}
	}
	return nil
}

// Children exposes the child validators for tree traversal (Walk, PrintRule).
func (a *andValidator) Children() []Validator {
	return a.validators
}

// Or returns a composite that succeeds when ANY child succeeds.
// It tries all children and returns an error only if every one fails.
func Or(validators ...Validator) Validator {
	return &orValidator{validators: validators}
}

type orValidator struct {
	validators []Validator
}

func (o *orValidator) Validate(value string) error {
	var errs []string
	for _, v := range o.validators {
		if err := v.Validate(value); err == nil {
			return nil
		} else {
			errs = append(errs, err.Error())
		}
	}
	return fmt.Errorf("none of %d rules passed: %s", len(o.validators), strings.Join(errs, "; "))
}

func (o *orValidator) Children() []Validator {
	return o.validators
}

// Not inverts a validator. msg is the error message when the inner validator
// succeeds (i.e., when the negation fails).
func Not(v Validator, msg string) Validator {
	return &notValidator{inner: v, msg: msg}
}

type notValidator struct {
	inner Validator
	msg   string
}

func (n *notValidator) Validate(value string) error {
	if err := n.inner.Validate(value); err == nil {
		return fmt.Errorf("%s", n.msg)
	}
	return nil
}

func (n *notValidator) Children() []Validator {
	return []Validator{n.inner}
}

// ---------------------------------------------------------------------------
// Leaf validators
// ---------------------------------------------------------------------------

// NonEmpty rejects empty strings.
func NonEmpty() Validator {
	return ValidatorFunc(func(value string) error {
		if value == "" {
			return fmt.Errorf("must not be empty")
		}
		return nil
	})
}

// MinLength rejects strings shorter than n.
func MinLength(n int) Validator {
	return ValidatorFunc(func(value string) error {
		if len(value) < n {
			return fmt.Errorf("length %d < minimum %d", len(value), n)
		}
		return nil
	})
}

// MaxLength rejects strings longer than n.
func MaxLength(n int) Validator {
	return ValidatorFunc(func(value string) error {
		if len(value) > n {
			return fmt.Errorf("length %d > maximum %d", len(value), n)
		}
		return nil
	})
}

// Matches rejects strings that don't match the given regexp pattern.
// Panics if pattern is invalid (programming error, same as regexp.MustCompile).
func Matches(pattern string) Validator {
	re := regexp.MustCompile(pattern)
	return ValidatorFunc(func(value string) error {
		if !re.MatchString(value) {
			return fmt.Errorf("%q does not match %s", value, pattern)
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Adapter: function → interface
// ---------------------------------------------------------------------------

// ValidatorFunc adapts a plain function to the Validator interface.
// This is the same pattern as http.HandlerFunc.
type ValidatorFunc func(string) error

func (f ValidatorFunc) Validate(value string) error { return f(value) }
