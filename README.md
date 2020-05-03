# jtd: JSON Validation for Golang

[![GoDoc Badge][badge]][godoc]

> This package implements JSON Typedef *validation* for Golang. If you're trying
> to do JSON Typedef *code generation*, see ["Generating Golang from JSON
> Typedef Schemas"][jtd-go-codegen] in the JSON Typedef docs.

`jtd` is a Golang implementation of [JSON Type Definition][jtd], a schema
language for JSON. `jtd` primarily gives you two things:

1. Validating input data against JSON Typedef schemas.
2. A Golang representation of JSON Typedef schemas.

With this package, you can add JSON Typedef-powered validation to your
application, or you can build your own tooling on top of JSON Type Definition.

## Installation

If you're using Go modules, install this package by running:

```bash
go get github.com/jsontypedef/json-typedef-go
```

Although the package's name ends in `json-typedef-go`, it exposes a package
called `jtd`. In other words, this:

```go
import "github.com/jsontypedef/json-typedef-go"
```

Is the same thing as:

```go
import jtd "github.com/jsontypedef/json-typedef-go"
```

## Documentation

Detailed API documentation is available online at:

https://godoc.org/github.com/jsontypedef/json-typedef-go

For more high-level documentation about JSON Typedef in general, or JSON Typedef
in combination with Golang in particular, see:

* [The JSON Typedef Website][jtd]
* ["Validating JSON in Golang with JSON Typedef"][jtd-go-validation]
* ["Generating Golang from JSON Typedef Schemas"][jtd-go-codegen]

## Basic Usage

> For a more detailed tutorial and guidance on how to integrate `jtd` in your
> application, see ["Validating JSON in Golang with JSON
> Typedef"][jtd-go-validation] in the JSON Typedef docs.

Here's an example of how you can use this package to validate JSON data against
a JSON Typedef schema:

```go
package main

import (
	"encoding/json"
	"fmt"

	jtd "github.com/jsontypedef/json-typedef-go"
)

func main() {
	schemaJSON := `{
		"properties": {
			"name": { "type": "string" },
			"age": { "type": "uint32" },
			"phones": {
				"elements": { "type": "string" }
			}
		}
	}`

	var schema jtd.Schema
	json.Unmarshal([]byte(schemaJSON), &schema)

	// jtd.Validate returns an array of validation errors. If there were no
	// problems with the input, it returns an empty array.

	// This input is perfect, so we'll get back an empty list of validation
	// errors.
	okJSON := `{
		"name": "John Doe",
		"age": 43,
		"phones": ["+44 1234567", "+44 2345678"]
	}`

	var ok interface{}
	json.Unmarshal([]byte(okJSON), &ok)

	// Outputs:
	// [] <nil>
	fmt.Println(jtd.Validate(schema, ok))

	// This next input has three problems with it:
	//
	// 1. It's missing "name", which is a required property.
	// 2. "age" is a string, but it should be an integer.
	// 3. "phones[1]" is a number, but it should be a string.
	//
	// Each of those errors corresponds to one of the errors returned by
	// jtd.Validate.
	badJSON := `{
		"age": "43",
		"phones": ["+44 1234567", 442345678]
	}`

	var bad interface{}
	json.Unmarshal([]byte(badJSON), &bad)

	// Outputs something like (order may change):
	//
	// []jtd.ValidateError{
	// 	jtd.ValidateError{
	// 		InstancePath: []string{},
	// 		SchemaPath: []string{"properties", "name"}
	// 	},
	// 	jtd.ValidateError{
	// 		InstancePath: []string{"age"},
	// 		SchemaPath: []string{"properties", "age", "type"}
	// 	},
	// 	jtd.ValidateError{
	// 		InstancePath: []string{"phones", "1"},
	// 		SchemaPath: []string{"properties", "phones", "elements", "type"}
	// 	}
	// }
	errs, _ := jtd.Validate(schema, bad)
	fmt.Printf("%#v\n", errs)
}
```

## Advanced Usage: Limiting Errors Returned

By default, `jtd.Validate` returns every error it finds. If you just care about
whether there are any errors at all, or if you can't show more than some number
of errors, then you can get better performance out of `jtd.Validate` using the
`WithMaxErrors` option.

For example, taking the same example from before, but limiting it to 1 error, we
get:

```ts
// []jtd.ValidateError{
// 	jtd.ValidateError{
// 		InstancePath: []string{},
// 		SchemaPath: []string{"properties", "name"}
// 	}
// }
errs, _ := jtd.Validate(schema, bad, jtd.WithMaxErrors(1))
fmt.Printf("%#v\n", errs)
```

## Advanced Usage: Handling Untrusted Schemas

If you want to run `jtd` against a schema that you don't trust, then you should:

1. Ensure the schema is well-formed, using the `Validate` method on `Schema`,
   which validates things like making sure all `ref`s have corresponding
   definitions.

2. Call `jtd.Validate` with the `WithMaxDepth` option. JSON Typedef lets you
   write recursive schemas -- if you're evaluating against untrusted schemas,
   you might go into an infinite loop when evaluating against a malicious input,
   such as this one:

   ```json
   {
     "ref": "loop",
     "definitions": {
       "loop": {
         "ref": "loop"
       }
     }
   }
   ```

   The `MaxDepth` option tells `jtd.Validate` how many `ref`s to follow
   recursively before giving up and throwing `jtd.ErrMaxDepthExceeded`.

Here's an example of how you can use `jtd` to evaluate data against an untrusted
schema:

```go
func validateUntrusted(schema jtd.Schema, instance interface{}) (bool, error) {
	if err := schema.Validate(); err != nil {
		return false, err
	}

	// You should tune WithMaxDepth to be high enough that most legitimate schemas
	// evaluate without errors, but low enough that an attacker cannot cause a
	// denial of service attack.
	errs, err := jtd.Validate(schema, instance, jtd.WithMaxDepth(32))
	if err != nil {
		return false, err
	}

	return len(errs) == 0, nil
}

// Returns true
validateUntrusted(jtd.Schema{Type: jtd.TypeString}, "foo")

// Returns false
validateUntrusted(jtd.Schema{Type: jtd.TypeString}, nil)

// Returns jtd.ErrInvalidType
validateUntrusted(jtd.Schema{Type: "nonsense"}, nil)

// Returns jtd.ErrMaxDepthExceeded
loop := "loop"
validateUntrusted(jtd.Schema{
	Definitions: map[string]jtd.Schema{
		"loop": jtd.Schema{
			Ref: &loop,
		},
	},
	Ref: &loop,
}, nil)
```

[badge]: https://godoc.org/github.com/jsontypedef/json-typedef-go?status.svg
[godoc]: https://godoc.org/github.com/jsontypedef/json-typedef-go
[jtd]: https://jsontypedef.com
[jtd-go-codegen]: https://jsontypedef.com/docs/go/code-generation
[jtd-go-validation]: https://jsontypedef.com/docs/go/validation
