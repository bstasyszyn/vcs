/*
Copyright Gen Digital Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package jsonschema

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

type ValidationErrors []gojsonschema.ResultError

func (e ValidationErrors) Error() string {
	var errMsg string

	for i, msg := range e {
		errMsg += msg.String()
		if i+1 < len(e) {
			errMsg += "; "
		}
	}

	return fmt.Sprintf("[%s]", errMsg)
}

type Validator interface {
	ValidateJSONSchema(data interface{}) error
}

type GoJSONSchemaValidator struct {
	schema *gojsonschema.Schema
}

func (v *GoJSONSchemaValidator) ValidateJSONSchema(data interface{}) error {
	result, err := v.schema.Validate(gojsonschema.NewGoLoader(data))
	if err != nil {
		return fmt.Errorf("loader error: %w", err)
	}

	if !result.Valid() {
		return fmt.Errorf("validation error: %w", ValidationErrors(result.Errors()))
	}

	return nil
}

type CachingValidator struct {
	cache map[string]Validator
	mutex sync.RWMutex
}

func NewCachingValidator() *CachingValidator {
	return &CachingValidator{cache: make(map[string]Validator)}
}

func (c *CachingValidator) Validate(data interface{}, schemaDoc map[string]interface{}) error {
	v, err := c.get(schemaDoc)
	if err != nil {
		return fmt.Errorf("get schema validator from cache: %w", err)
	}

	return v.ValidateJSONSchema(data)
}

func (c *CachingValidator) get(schema map[string]interface{}) (Validator, error) {
	schemaIDObj, ok := schema["$id"]
	if !ok {
		return nil, fmt.Errorf("field '$id' not found in JSON schema")
	}

	schemaID, ok := schemaIDObj.(string)
	if !ok {
		return nil, fmt.Errorf("expecting field '$id' in JSON schema to be a string type but was %s",
			reflect.TypeOf(schemaIDObj))
	}

	c.mutex.RLock()
	v, ok := c.cache[schemaID]
	c.mutex.RUnlock()

	if ok {
		return v, nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	schemaValidator, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewGoLoader(schema))
	if err != nil {
		return nil, fmt.Errorf("compile JSON schema [%s]: %w", schemaID, err)
	}

	return &GoJSONSchemaValidator{schema: schemaValidator}, nil
}
