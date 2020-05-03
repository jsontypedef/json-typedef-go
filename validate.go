package jtd

import (
	"errors"
	"math"
	"strconv"
	"time"
)

// ValidateSettings are settings that configure ValidateWithSettings.
type ValidateSettings struct {
	// The maximum number of refs to recursively follow before returning
	// ErrMaxDepthExceeded. Zero disables a max depth altogether.
	MaxDepth int

	// The maximum number of validation errors to return. Zero disables a max
	// number of errors altogether.
	MaxErrors int
}

// ValidateOption is an option you can pass to Validate.
type ValidateOption func(*ValidateSettings)

// WithMaxDepth sets the the MaxDepth option of ValidateSettings.
func WithMaxDepth(maxDepth int) ValidateOption {
	return func(settings *ValidateSettings) {
		settings.MaxDepth = maxDepth
	}
}

// WithMaxErrors sets the the MaxErrors option of ValidateSettings.
func WithMaxErrors(maxErrors int) ValidateOption {
	return func(settings *ValidateSettings) {
		settings.MaxErrors = maxErrors
	}
}

// ValidateError is a validation error returned from Validate.
//
// This corresponds to a standard error indicator from the JSON Typedef
// specification.
type ValidateError struct {
	// Path to the part of the instance that was invalid.
	InstancePath []string

	// Path to the part of the schema that rejected the instance.
	SchemaPath []string
}

// ErrMaxDepthExceeded is the error returned from Validate if too many refs are
// recursively followed.
//
// The maximum depth of refs to follow is controlled by MaxErrors in
// ValidateSettings.
var ErrMaxDepthExceeded = errors.New("jtd: max depth exceeded")

// Validate validates a schema against an instance (or "input").
//
// Returns ErrMaxDepthExceeded if too many refs are recursively followed while
// validating. Otherwise, returns a set of ValidateError, in conformance with
// the JSON Typedef specification.
func Validate(schema Schema, instance interface{}, opts ...ValidateOption) ([]ValidateError, error) {
	settings := ValidateSettings{}
	for _, opt := range opts {
		opt(&settings)
	}

	return ValidateWithSettings(settings, schema, instance)
}

// ValidateWithSettings validates a schema against an instance, using a set of
// settings.
//
// Returns ErrMaxDepthExceeded if too many refs are recursively followed while
// validating. Otherwise, returns a set of ValidateError, in conformance with
// the JSON Typedef specification.
func ValidateWithSettings(settings ValidateSettings, schema Schema, instance interface{}) ([]ValidateError, error) {
	state := validateState{
		Errors:         []ValidateError{},
		InstanceTokens: []string{},
		SchemaTokens:   [][]string{[]string{}},
		Root:           schema,
		Settings:       settings,
	}

	// errMaxErrorsReached is just an internal error used to quickly abort further
	// validation. It is not an actual error for the end user, just a
	// circuit-breaker used by validate internally.
	if err := validate(&state, schema, instance, nil); err != nil && err != errMaxErrorsReached {
		return nil, err
	}

	return state.Errors, nil
}

func validate(state *validateState, schema Schema, instance interface{}, parentTag *string) error {
	if schema.Nullable && instance == nil {
		return nil
	}

	switch schema.Form() {
	case FormEmpty:
		return nil
	case FormRef:
		if len(state.SchemaTokens) == state.Settings.MaxDepth {
			return ErrMaxDepthExceeded
		}

		state.SchemaTokens = append(state.SchemaTokens, []string{"definitions", *schema.Ref})
		if err := validate(state, state.Root.Definitions[*schema.Ref], instance, nil); err != nil {
			return err
		}
		state.SchemaTokens = state.SchemaTokens[:len(state.SchemaTokens)-1]
	case FormType:
		state.pushSchemaToken("type")

		switch schema.Type {
		case TypeBoolean:
			if _, ok := instance.(bool); !ok {
				if err := state.pushError(); err != nil {
					return err
				}
			}
		case TypeFloat32, TypeFloat64:
			if _, ok := instance.(float64); !ok {
				if err := state.pushError(); err != nil {
					return err
				}
			}
		case TypeInt8:
			if err := validateInt(state, instance, -128.0, 127.0); err != nil {
				return err
			}
		case TypeUint8:
			if err := validateInt(state, instance, 0.0, 255.0); err != nil {
				return err
			}
		case TypeInt16:
			if err := validateInt(state, instance, -32768.0, 32767.0); err != nil {
				return err
			}
		case TypeUint16:
			if err := validateInt(state, instance, 0.0, 65535.0); err != nil {
				return err
			}
		case TypeInt32:
			if err := validateInt(state, instance, -2147483648.0, 2147483647.0); err != nil {
				return err
			}
		case TypeUint32:
			if err := validateInt(state, instance, 0.0, 4294967295.0); err != nil {
				return err
			}
		case TypeString:
			if _, ok := instance.(string); !ok {
				if err := state.pushError(); err != nil {
					return err
				}
			}
		case TypeTimestamp:
			if s, ok := instance.(string); ok {
				if _, err := time.Parse(time.RFC3339, s); err != nil {
					if err := state.pushError(); err != nil {
						return err
					}
				}
			} else {
				if err := state.pushError(); err != nil {
					return err
				}
			}
		}

		state.popSchemaToken()
	case FormEnum:
		state.pushSchemaToken("enum")
		if s, ok := instance.(string); ok {
			ok := false
			for _, value := range schema.Enum {
				if s == value {
					ok = true
				}
			}

			if !ok {
				if err := state.pushError(); err != nil {
					return err
				}
			}
		} else {
			if err := state.pushError(); err != nil {
				return err
			}
		}
		state.popSchemaToken()
	case FormElements:
		state.pushSchemaToken("elements")
		if arr, ok := instance.([]interface{}); ok {
			for i, subInstance := range arr {
				state.pushInstanceToken(strconv.Itoa(i))
				if err := validate(state, *schema.Elements, subInstance, nil); err != nil {
					return err
				}
				state.popInstanceToken()
			}
		} else {
			if err := state.pushError(); err != nil {
				return err
			}
		}
		state.popSchemaToken()
	case FormProperties:
		if obj, ok := instance.(map[string]interface{}); ok {
			state.pushSchemaToken("properties")
			for key, subSchema := range schema.Properties {
				state.pushSchemaToken(key)
				if subInstance, ok := obj[key]; ok {
					state.pushInstanceToken(key)
					if err := validate(state, subSchema, subInstance, nil); err != nil {
						return err
					}
					state.popInstanceToken()
				} else {
					if err := state.pushError(); err != nil {
						return err
					}
				}
				state.popSchemaToken()
			}
			state.popSchemaToken()

			state.pushSchemaToken("optionalProperties")
			for key, subSchema := range schema.OptionalProperties {
				state.pushSchemaToken(key)
				if subInstance, ok := obj[key]; ok {
					state.pushInstanceToken(key)
					if err := validate(state, subSchema, subInstance, nil); err != nil {
						return err
					}
					state.popInstanceToken()
				}
				state.popSchemaToken()
			}
			state.popSchemaToken()

			if !schema.AdditionalProperties {
				for key := range obj {
					if parentTag != nil && key == *parentTag {
						continue
					}

					requiredOk := false
					optionalOk := false

					if schema.Properties != nil {
						_, requiredOk = schema.Properties[key]
					}

					if schema.OptionalProperties != nil {
						_, optionalOk = schema.OptionalProperties[key]
					}

					if !requiredOk && !optionalOk {
						state.pushInstanceToken(key)
						if err := state.pushError(); err != nil {
							return err
						}
						state.popInstanceToken()
					}
				}
			}
		} else {
			if schema.Properties != nil {
				state.pushSchemaToken("properties")
			} else {
				state.pushSchemaToken("optionalProperties")
			}

			if err := state.pushError(); err != nil {
				return err
			}

			state.popSchemaToken()
		}
	case FormValues:
		state.pushSchemaToken("values")
		if obj, ok := instance.(map[string]interface{}); ok {
			for key, subInstance := range obj {
				state.pushInstanceToken(key)
				if err := validate(state, *schema.Values, subInstance, nil); err != nil {
					return err
				}
				state.popInstanceToken()
			}
		} else {
			if err := state.pushError(); err != nil {
				return err
			}
		}
		state.popSchemaToken()
	case FormDiscriminator:
		if obj, ok := instance.(map[string]interface{}); ok {
			if tag, ok := obj[schema.Discriminator]; ok {
				if tagStr, ok := tag.(string); ok {
					if mapping, ok := schema.Mapping[tagStr]; ok {
						state.pushSchemaToken("mapping")
						state.pushSchemaToken(tagStr)

						if err := validate(state, mapping, instance, &schema.Discriminator); err != nil {
							return err
						}

						state.popSchemaToken()
						state.popSchemaToken()
					} else {
						state.pushSchemaToken("mapping")
						state.pushInstanceToken(schema.Discriminator)
						if err := state.pushError(); err != nil {
							return err
						}
						state.popInstanceToken()
						state.popSchemaToken()
					}
				} else {
					state.pushSchemaToken("discriminator")
					state.pushInstanceToken(schema.Discriminator)
					if err := state.pushError(); err != nil {
						return err
					}
					state.popInstanceToken()
					state.popSchemaToken()
				}
			} else {
				state.pushSchemaToken("discriminator")
				if err := state.pushError(); err != nil {
					return err
				}
				state.popSchemaToken()
			}
		} else {
			state.pushSchemaToken("discriminator")
			if err := state.pushError(); err != nil {
				return err
			}
			state.popSchemaToken()
		}
	}

	return nil
}

func validateInt(state *validateState, instance interface{}, min, max float64) error {
	if n, ok := instance.(float64); ok {
		if i, f := math.Modf(n); f != 0.0 || i < min || i > max {
			if err := state.pushError(); err != nil {
				return err
			}
		}
	} else {
		if err := state.pushError(); err != nil {
			return err
		}
	}

	return nil
}

var errMaxErrorsReached = errors.New("jtd internal: max errors reached")

type validateState struct {
	Errors         []ValidateError
	InstanceTokens []string
	SchemaTokens   [][]string
	Root           Schema
	Settings       ValidateSettings
}

func (vs *validateState) pushInstanceToken(token string) {
	vs.InstanceTokens = append(vs.InstanceTokens, token)
}

func (vs *validateState) popInstanceToken() {
	vs.InstanceTokens = vs.InstanceTokens[:len(vs.InstanceTokens)-1]
}

func (vs *validateState) pushSchemaToken(token string) {
	vs.SchemaTokens[len(vs.SchemaTokens)-1] = append(vs.SchemaTokens[len(vs.SchemaTokens)-1], token)
}

func (vs *validateState) popSchemaToken() {
	last := vs.SchemaTokens[len(vs.SchemaTokens)-1]
	vs.SchemaTokens[len(vs.SchemaTokens)-1] = last[:len(last)-1]
}

func (vs *validateState) pushError() error {
	instanceTokens := make([]string, len(vs.InstanceTokens))
	copy(instanceTokens, vs.InstanceTokens)

	schemaTokens := make([]string, len(vs.SchemaTokens[len(vs.SchemaTokens)-1]))
	copy(schemaTokens, vs.SchemaTokens[len(vs.SchemaTokens)-1])

	vs.Errors = append(vs.Errors, ValidateError{
		InstancePath: instanceTokens,
		SchemaPath:   schemaTokens,
	})

	if len(vs.Errors) == vs.Settings.MaxErrors {
		return errMaxErrorsReached
	}

	return nil
}
