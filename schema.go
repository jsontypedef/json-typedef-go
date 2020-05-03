package jtd

import (
	"errors"
	"reflect"
)

// Schema represents a JSON Typedef Schema.
type Schema struct {
	Definitions          map[string]Schema      `json:"definitions"`
	Metadata             map[string]interface{} `json:"metadata"`
	Nullable             bool                   `json:"nullable"`
	Ref                  *string                `json:"ref"`
	Type                 Type                   `json:"type"`
	Enum                 []string               `json:"enum"`
	Elements             *Schema                `json:"elements"`
	Properties           map[string]Schema      `json:"properties"`
	OptionalProperties   map[string]Schema      `json:"optionalProperties"`
	AdditionalProperties bool                   `json:"additionalProperties"`
	Values               *Schema                `json:"values"`
	Discriminator        string                 `json:"discriminator"`
	Mapping              map[string]Schema      `json:"mapping"`
}

// Type represents the values that the JSON Typedef "type" keyword can take on.
type Type string

const (
	// TypeBoolean represents true or false.
	TypeBoolean Type = "boolean"

	// TypeFloat32 represents a JSON number. Code generators will create a
	// single-precision floating point from this.
	TypeFloat32 = "float32"

	// TypeFloat64 represents a JSON number. Code generators will create a
	// double-precision floating point from this.
	TypeFloat64 = "float64"

	// TypeInt8 represents a JSON number within the range of a int8.
	TypeInt8 = "int8"

	// TypeUint8 represents a JSON number within the range of a uint8.
	TypeUint8 = "uint8"

	// TypeInt16 represents a JSON number within the range of a int16.
	TypeInt16 = "int16"

	// TypeUint16 represents a JSON number within the range of a uint16.
	TypeUint16 = "uint16"

	// TypeInt32 represents a JSON number within the range of a int32.
	TypeInt32 = "int32"

	// TypeUint32 represents a JSON number within the range of a uint32.
	TypeUint32 = "uint32"

	// TypeString represents a JSON string.
	TypeString = "string"

	// TypeTimestamp represents a JSON string containing a RFC3339 timestamp.
	TypeTimestamp = "timestamp"
)

// ErrInvalidForm indicates that a schema uses an invalid combination of
// keywords.
var ErrInvalidForm = errors.New("jtd: invalid form")

// ErrNonRootDefinition indicates that a schema uses the "definition" keyword
// outside of a root schema.
var ErrNonRootDefinition = errors.New("jtd: non-root definitions")

// ErrNoSuchDefinition indicates that a schema has a "ref" with no corresponding
// definition.
var ErrNoSuchDefinition = errors.New("jtd: ref to non-existent definition")

// ErrInvalidType indicates that a schema has a "type" keyword with an invalid
// value.
var ErrInvalidType = errors.New("jtd: invalid type")

// ErrEmptyEnum indicates that a schema has a "enum" keyword with no values.
var ErrEmptyEnum = errors.New("jtd: empty enum")

// ErrRepeatedEnumValue indicates that a schema has a "enum" keyword with
// repeated values.
var ErrRepeatedEnumValue = errors.New("jtd: enum contains repeated values")

// ErrSharedProperty indicates that a schema has the same property name in
// "properties" and "optionalProperties".
var ErrSharedProperty = errors.New("jtd: properties and optionalProperties share property")

// ErrNonPropertiesMapping indicates that a schema has a mapping value that
// isn't a schema of the properties form.
var ErrNonPropertiesMapping = errors.New("jtd: mapping value not of properties form")

// ErrMappingRepeatedDiscriminator indicates that a schema has a mapping value
// that has the same property as the discriminator it's within.
var ErrMappingRepeatedDiscriminator = errors.New("jtd: mapping re-specifies discriminator property")

// ErrNullableMapping indicates that a schema has a mapping value with
// "nullable" set to true.
var ErrNullableMapping = errors.New("jtd: mapping allows for nullable values")

// Validate returns an error if a schema is not a valid root JSON Typedef
// schema.
//
// Validate may return one of ErrInvalidForm, ErrNonRootDefinition,
// ErrNoSuchDefinition, ErrInvalidType, ErrEmptyEnum, ErrRepeatedEnumValue,
// ErrSharedProperty, ErrNonPropertiesMapping, ErrMappingRepeatedDiscriminator,
// or ErrNullableMapping.
func (s Schema) Validate() error {
	return s.ValidateWithRoot(true, s)
}

// Index of valid form "signatures" -- i.e., combinations of the presence of the
// keywords (in order):
//
// ref type enum elements properties optionalProperties additionalProperties
// values discriminator mapping
//
// The keywords "definitions", "nullable", and "metadata" are not included here,
// because they would restrict nothing.
var validForms = [][]bool{
	// Empty form
	{false, false, false, false, false, false, false, false, false, false},
	// Ref form
	{true, false, false, false, false, false, false, false, false, false},
	// Type form
	{false, true, false, false, false, false, false, false, false, false},
	// Enum form
	{false, false, true, false, false, false, false, false, false, false},
	// Elements form
	{false, false, false, true, false, false, false, false, false, false},
	// Properties form -- properties or optional properties or both, and never
	// additional properties on its own
	{false, false, false, false, true, false, false, false, false, false},
	{false, false, false, false, false, true, false, false, false, false},
	{false, false, false, false, true, true, false, false, false, false},
	{false, false, false, false, true, false, true, false, false, false},
	{false, false, false, false, false, true, true, false, false, false},
	{false, false, false, false, true, true, true, false, false, false},
	// Values form
	{false, false, false, false, false, false, false, true, false, false},
	// Discriminator form
	{false, false, false, false, false, false, false, false, true, true},
}

// ValidateWithRoot returns an error if s is not a valid schema, given the root
// schema s is supposed to appear within.
//
// isRoot indicates whether the schema is expected to be a root schema. root is
// the root schema s is supposed to be contained within. If isRoot is true, then
// root should be equal to s for the return value to be meaningful.
func (s Schema) ValidateWithRoot(isRoot bool, root Schema) error {
	formSignature := []bool{
		s.Ref != nil,
		s.Type != "",
		s.Enum != nil,
		s.Elements != nil,
		s.Properties != nil,
		s.OptionalProperties != nil,
		s.AdditionalProperties,
		s.Values != nil,
		s.Discriminator != "",
		s.Mapping != nil,
	}

	formOk := false
	for _, form := range validForms {
		formOk = formOk || reflect.DeepEqual(formSignature, form)
	}

	if !formOk {
		return ErrInvalidForm
	}

	if s.Definitions != nil && !isRoot {
		return ErrNonRootDefinition
	}

	for _, s := range s.Definitions {
		if err := s.ValidateWithRoot(false, root); err != nil {
			return err
		}
	}

	if s.Ref != nil {
		if root.Definitions == nil {
			return ErrNoSuchDefinition
		}

		if _, ok := root.Definitions[*s.Ref]; !ok {
			return ErrNoSuchDefinition
		}
	}

	if s.Type != "" {
		validTypes := []Type{
			TypeBoolean,
			TypeFloat32,
			TypeFloat64,
			TypeInt8,
			TypeUint8,
			TypeInt16,
			TypeUint16,
			TypeInt32,
			TypeUint32,
			TypeString,
			TypeTimestamp,
		}

		ok := false
		for _, t := range validTypes {
			if s.Type == t {
				ok = true
			}
		}

		if !ok {
			return ErrInvalidType
		}
	}

	if s.Enum != nil {
		if len(s.Enum) == 0 {
			return ErrEmptyEnum
		}

		dedupe := map[string]struct{}{}
		for _, value := range s.Enum {
			if _, ok := dedupe[value]; ok {
				return ErrRepeatedEnumValue
			}

			dedupe[value] = struct{}{}
		}
	}

	if s.Elements != nil {
		if err := s.Elements.ValidateWithRoot(false, root); err != nil {
			return err
		}
	}

	for k, p := range s.Properties {
		if err := p.ValidateWithRoot(false, root); err != nil {
			return err
		}

		if s.OptionalProperties != nil {
			if _, ok := s.OptionalProperties[k]; ok {
				return ErrSharedProperty
			}
		}
	}

	for _, s := range s.OptionalProperties {
		if err := s.ValidateWithRoot(false, root); err != nil {
			return err
		}
	}

	if s.Values != nil {
		if err := s.Values.ValidateWithRoot(false, root); err != nil {
			return err
		}
	}

	for _, m := range s.Mapping {
		if err := m.ValidateWithRoot(false, root); err != nil {
			return err
		}

		if m.Form() != FormProperties {
			return ErrNonPropertiesMapping
		}

		if m.Properties != nil {
			if _, ok := m.Properties[s.Discriminator]; ok {
				return ErrMappingRepeatedDiscriminator
			}
		}

		if m.OptionalProperties != nil {
			if _, ok := m.OptionalProperties[s.Discriminator]; ok {
				return ErrMappingRepeatedDiscriminator
			}
		}

		if m.Nullable {
			return ErrNullableMapping
		}
	}

	return nil
}

// Form returns JSON Typedef schema form that s takes on.
func (s Schema) Form() Form {
	if s.Ref != nil {
		return FormRef
	}

	if s.Type != "" {
		return FormType
	}

	if s.Enum != nil {
		return FormEnum
	}

	if s.Elements != nil {
		return FormElements
	}

	if s.Properties != nil || s.OptionalProperties != nil {
		return FormProperties
	}

	if s.Values != nil {
		return FormValues
	}

	if s.Mapping != nil {
		return FormDiscriminator
	}

	return FormEmpty
}

// Form is an enumeration of the eight forms a JSON Typedef schema may take on.
type Form string

const (
	// FormEmpty is the empty form.
	FormEmpty Form = "empty"

	// FormRef is the ref form.
	FormRef = "ref"

	// FormType is the type form.
	FormType = "type"

	// FormEnum is the enum form.
	FormEnum = "enum"

	// FormElements is the elements form.
	FormElements = "elements"

	// FormProperties is the properties form.
	FormProperties = "properties"

	// FormValues is the values form.
	FormValues = "values"

	// FormDiscriminator is the discriminator form.
	FormDiscriminator = "discriminator"
)
