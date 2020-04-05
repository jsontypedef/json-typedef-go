package jtd_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	jtd "github.com/jsontypedef/json-typedef-go"
	"github.com/stretchr/testify/assert"
)

func TestInvalidSchemas(t *testing.T) {
	spec, err := ioutil.ReadFile("json-typedef-spec/tests/invalid_schemas.json")
	assert.NoError(t, err)

	var testCases map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(spec, &testCases))

	for name, invalidSchema := range testCases {
		t.Run(name, func(t *testing.T) {
			// The encoding/json package will decode "null" into the zero value of
			// Schema, which is a valid schema. There isn't a straightforward, Go-like
			// way to detect and prevent a schema from being null.
			if string(invalidSchema) == "null" {
				return
			}

			decoder := json.NewDecoder(bytes.NewBuffer(invalidSchema))
			decoder.DisallowUnknownFields()

			var schema jtd.Schema
			if err := decoder.Decode(&schema); err != nil {
				return
			}

			assert.Error(t, schema.Validate())
		})
	}
}
