package jtd_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	jtd "github.com/jsontypedef/json-typedef-go"
	"github.com/stretchr/testify/assert"
)

func TestMaxDepth(t *testing.T) {
	foo := "foo"
	schema := jtd.Schema{
		Definitions: map[string]jtd.Schema{
			"foo": jtd.Schema{Ref: &foo},
		},
		Ref: &foo,
	}

	_, err := jtd.Validate(schema, nil, jtd.WithMaxDepth(3))
	assert.Equal(t, jtd.ErrMaxDepthExceeded, err)
}

func TestMaxErrors(t *testing.T) {
	schema := jtd.Schema{
		Elements: &jtd.Schema{
			Type: jtd.TypeBoolean,
		},
	}

	instance := []interface{}{nil, nil, nil, nil, nil}

	res, err := jtd.Validate(schema, instance, jtd.WithMaxErrors(3))
	assert.NoError(t, err)
	assert.Equal(t, 3, len(res))
}

type testCase struct {
	Schema   jtd.Schema  `json:"schema"`
	Instance interface{} `json:"instance"`
	Errors   []struct {
		InstancePath []string `json:"instancePath"`
		SchemaPath   []string `json:"schemaPath"`
	} `json:"errors"`
}

// Tests we knowingly do not support from the test suite.
var skippedTests = []string{
	// We skip two tests related to timestamps, because the stdlib's time package
	// does not support leap seconds in RFC3339 timestamps.
	"timestamp type schema - 1990-12-31T23:59:60Z",
	"timestamp type schema - 1990-12-31T15:59:60-08:00",
}

func TestValidation(t *testing.T) {
	spec, err := ioutil.ReadFile("json-typedef-spec/tests/validation.json")
	assert.NoError(t, err)

	var testCases map[string]testCase
	assert.NoError(t, json.Unmarshal(spec, &testCases))

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, skippedTest := range skippedTests {
				if name == skippedTest {
					t.Skip()
				}
			}

			expectedErrors := []jtd.ValidateError{}
			for _, e := range tt.Errors {
				expectedErrors = append(expectedErrors, jtd.ValidateError{
					InstancePath: e.InstancePath,
					SchemaPath:   e.SchemaPath,
				})
			}

			assert.NoError(t, tt.Schema.Validate())
			validateErrors, err := jtd.Validate(tt.Schema, tt.Instance)
			assert.NoError(t, err)

			sort.Slice(validateErrors, func(i, j int) bool {
				a0 := strings.Join(validateErrors[i].SchemaPath, "/")
				b0 := strings.Join(validateErrors[j].SchemaPath, "/")
				a1 := strings.Join(validateErrors[i].InstancePath, "/")
				b1 := strings.Join(validateErrors[j].InstancePath, "/")

				return (a0 + ":" + a1) < (b0 + ":" + b1)
			})

			sort.Slice(expectedErrors, func(i, j int) bool {
				a0 := strings.Join(expectedErrors[i].SchemaPath, "/")
				b0 := strings.Join(expectedErrors[j].SchemaPath, "/")
				a1 := strings.Join(expectedErrors[i].InstancePath, "/")
				b1 := strings.Join(expectedErrors[j].InstancePath, "/")

				return (a0 + ":" + a1) < (b0 + ":" + b1)
			})

			assert.Equal(t, expectedErrors, validateErrors)
		})
	}
}

func ExampleValidate() {
	var schema jtd.Schema
	json.Unmarshal([]byte(`{
		"properties": {
			"name": { "type": "string" },
			"age": { "type": "uint32" },
			"phones": {
				"elements": { "type": "string" }
			}
		}
	}`), &schema)

	var dataOk interface{}
	json.Unmarshal([]byte(`{
		"name": "John Doe",
		"age": 43,
		"phones": ["+44 1234567", "+44 2345678"]
	}`), &dataOk)

	fmt.Println(jtd.Validate(schema, dataOk))

	var dataBad interface{}
	json.Unmarshal([]byte(`{
		"name": "John Doe",
		"age": 43,
		"phones": ["+44 1234567", 442345678]
	}`), &dataBad)

	fmt.Println(jtd.Validate(schema, dataBad))

	// Output:
	// [] <nil>
	// [{[phones 1] [properties phones elements type]}] <nil>
}

func ExampleValidate_maxDepth() {
	loop := "loop"
	schema := jtd.Schema{
		Definitions: map[string]jtd.Schema{
			"loop": jtd.Schema{
				Ref: &loop,
			},
		},
		Ref: &loop,
	}

	// If you ran this, you would overflow the stack:
	// jtd.Validate(schema, nil)

	fmt.Println(jtd.Validate(schema, nil, jtd.WithMaxDepth(32)))
	// Output:
	// [] jtd: max depth exceeded
}

func ExampleValidate_maxErrors() {
	schema := jtd.Schema{
		Elements: &jtd.Schema{
			Type: jtd.TypeBoolean,
		},
	}

	instance := []interface{}{nil, nil, nil, nil, nil}

	fmt.Println(jtd.Validate(schema, instance))
	fmt.Println(jtd.Validate(schema, instance, jtd.WithMaxErrors(3)))
	// Output:
	// [{[0] [elements type]} {[1] [elements type]} {[2] [elements type]} {[3] [elements type]} {[4] [elements type]}] <nil>
	// [{[0] [elements type]} {[1] [elements type]} {[2] [elements type]}] <nil>
}
