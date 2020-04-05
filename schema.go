package jtd

import (
	"errors"
	"reflect"
)

type Schema struct {
	Definitions          map[string]Schema
	Metadata             map[string]interface{}
	Nullable             bool
	Ref                  *string
	Type                 Type
	Enum                 []string
	Elements             *Schema
	Properties           map[string]Schema
	OptionalProperties   map[string]Schema
	AdditionalProperties *bool
	Values               *Schema
	Discriminator        string
	Mapping              map[string]Schema
}

type Type string

const (
	TypeBoolean   Type = "boolean"
	TypeFloat32        = "float32"
	TypeFloat64        = "float64"
	TypeInt8           = "int8"
	TypeUint8          = "uint8"
	TypeInt16          = "int16"
	TypeUint16         = "uint16"
	TypeInt32          = "int32"
	TypeUint32         = "uint32"
	TypeString         = "string"
	TypeTimestamp      = "timestamp"
)

var ErrInvalidForm = errors.New("jtd: invalid form")
var ErrNonRootDefinition = errors.New("jtd: non-root definitions")
var ErrNoSuchDefinition = errors.New("jtd: ref to non-existent definition")
var ErrInvalidType = errors.New("jtd: invalid type")
var ErrEmptyEnum = errors.New("jtd: empty enum")
var ErrRepeatedEnumValue = errors.New("jtd: enum contains repeated values")
var ErrSharedProperty = errors.New("jtd: properties and optionalProperties share property")
var ErrNonPropertiesMapping = errors.New("jtd: mapping value not of properties form")
var ErrMappingRepeatedDiscriminator = errors.New("jtd: mapping re-specifies discriminator property")
var ErrNullableMapping = errors.New("jtd: mapping allows for nullable values")

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

func (s Schema) ValidateWithRoot(isRoot bool, root Schema) error {
	formSignature := []bool{
		s.Ref != nil,
		s.Type != "",
		s.Enum != nil,
		s.Elements != nil,
		s.Properties != nil,
		s.OptionalProperties != nil,
		s.AdditionalProperties != nil,
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

type Form string

const (
	FormEmpty         Form = "empty"
	FormRef                = "ref"
	FormType               = "type"
	FormEnum               = "enum"
	FormElements           = "elements"
	FormProperties         = "properties"
	FormValues             = "values"
	FormDiscriminator      = "discriminator"
)
