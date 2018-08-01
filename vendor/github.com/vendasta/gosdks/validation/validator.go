// Package validation provides the ability to define a Validator with one or
// more Rules, which can be checked by calling Validate().
//
// If any of the rules fail validation, Validate() will return an error:
//
//   err := NewValidator().Rule(
//       StringNotEmpty(businessID),
//       Optional(someURL != "", ValidURL(url, util.InvalidArgument, "Non-empty URL must be valid")),
//   ).Validate()
//   if err != nil {
//       return err
//   }
//
// Validators can also be chained together with Rule():
//
//   err := NewValidator().
//       Rule(StringNotEmpty(businessID)).
//       Rule(ListingIDRequired(listingID)).
//       Rule(PageSizeWithinBounds(pageSize)).
//       Validate()
//   if err != nil {
//       return err
//   }
//
package validation

import (
	"errors"
	"strings"
)

// Rule is a struct whose state can be evaluated with a single function.
//
// Consider adding any values that need to be validated as properties on your own Rule.
type Rule interface {
	Validate() error
}

// Validator provides a mechanism for chaining individual validation Rules together.
//
// Rules are evaluated in the same order that they were added to the Validator.
type Validator struct {
	rules []Rule
}

// NewValidator returns a fresh Validator devoid of any rules.
//
// Use this to create a new set of rules to validate like this:
//
//   err := NewValidator().
//       Rule(ListingIDRequired(listingID)).
//       Rule(PageSizeWithinBounds(pageSize)).
//       Validate()
//   if err != nil {
//       return err
//   }
func NewValidator() *Validator {
	return &Validator{}
}

// Rule adds one or more Rules to a Validator.
func (c *Validator) Rule(r ...Rule) *Validator {
	c.rules = append(c.rules, r...)
	return c
}

// Validates each rule in order and returns the first validaton error encountered, or nil if valid.
func (c *Validator) Validate() error {
	var err error
	for _, r := range c.rules {
		err = r.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateAndJoinErrors processes the rules in order and combines and returns all validation
// errors, or nil if valid.
func (c *Validator) ValidateAndJoinErrors() error {
	messages := []string{}
	var err error
	for _, r := range c.rules {
		err = r.Validate()
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) > 0 {
		return errors.New(strings.Join(messages, "\n"))
	}
	return nil
}
