package validator

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func TestDataModelValidator(t *testing.T) {
	model1 := `{
		"product": "Car",
		"code": 123,
		"info": {
			"price": 123.22,
			"count": 100,
			"total": 100.00,
			"expiration": "2033-03-03"
		}
	}`
	doc := jsoniter.Get([]byte(model1)).GetInterface()
	require.NotNil(t, doc)

	validator1, err1 := NewDataModelValidator("test1", `{ 
		"type": "object",
		"properties": {
			"product": { "type": "string" },
			"code": { "type": "number" },
			"info": { 
				"type": "object",
				"properties": {
					"price" : {"type": "number"},
					"count" : {"type": "integer"},
					"total" : {"type": "number"},
					"expiration" : {"type": "string", "format" : "date"}
				},
				"dependencies": {
					"count": ["total"]
				},
				"required" : ["price", "expiration"]
			}
		},
		"required" : ["product",  "code", "info"]             
	}`, "")
	require.NotNil(t, validator1)
	require.NoError(t, err1)
	require.Error(t, validator1.Validate(nil))
	require.Error(t, validator1.Validate("123"))
	require.NoError(t, validator1.Validate(doc))

	model20 := `{
		"name": "Musk",
		"age": 46,
	}`
	model21 := `{
		"name": "Musk",
	}`
	model22 := `{
		"name": "Musk",
		"address": "Unknown",
	}`
	doc20 := jsoniter.Get([]byte(model20)).GetInterface()
	doc21 := jsoniter.Get([]byte(model21)).GetInterface()
	doc22 := jsoniter.Get([]byte(model22)).GetInterface()
	require.NotNil(t, doc)
	validator2, err2 := NewDataModelValidator("test2", `{ 
		"type": "object",     
		"properties": {      
			"name": {"type" : "string"},
			"age" : {"type" : "integer"}
		},
		"required" : ["name"],
		"additionalProperties" : false
	}`, "")
	require.NotNil(t, validator2)
	require.NoError(t, err2)
	require.NoError(t, validator2.Validate(doc20))
	require.NoError(t, validator2.Validate(doc21))
	require.Error(t, validator2.Validate(doc22))

	model3 := "10010"
	validator3, err3 := NewDataModelValidator("model3", `{
		"type": "integer",
		"multipleOf": 10
	}`, "")
	require.NotNil(t, validator3)
	require.NoError(t, err3)
	require.Error(t, validator3.Validate("123"))
	require.NoError(t, validator3.Validate(1230))
	require.Error(t, validator3.Validate(model3))

	type Guest struct {
		Name string
		Age  uint
		City string
	}

	model4 := &Guest{
		Name: "Mr. Lao6",
		Age:  22,
		City: "Montrel",
	}
	validator4, err4 := NewDataModelValidator("model4", `{ 
		"type": "object"
	}`, `{
		"name": "IsFromMontrel",
		"desc": "Only allow those people from Montrel.",
		"when": "Result.IsValid && model4.City != \"Montrel\"",
		"then": [
			"Result.IsValid = false",
			"Result.Reason = \"this guy is not from Montrel\""
		]
	}`)
	require.NotNil(t, validator4)
	require.NoError(t, err4)
	require.NoError(t, validator4.Validate(model4))

	model4.City = "Lisbon"
	require.Error(t, validator4.Validate(model4))

	model50 := `{
		"billing_address": {
			"street_address": "No. 1 Street",
			"city": "Lonton"
		},
		"shipping_address": {
			"street_address": "No. 2 Street",
			"city": "Huston",
			"state" "Texas"
		}
	}`
	doc50 := jsoniter.Get([]byte(model50)).GetInterface()
	require.NotNil(t, doc)
	validator5, err5 := NewDataModelValidator("test5", `{
		"definitions": {
			"address": {
				"type": "object",
				"properties": {
					"street_address": { "type": "string" },
					"city":           { "type": "string" },
					"state":          { "type": "string" }
				},
				"required": ["street_address", "city"]
			}
		},
		"type": "object",
		"properties": {
			"billing_address": { "$ref": "#/definitions/address" },
			"shipping_address": { "$ref": "#/definitions/address" }
		}
	}`, "")
	require.NotNil(t, validator5)
	require.NoError(t, err5)
	require.NoError(t, validator5.Validate(doc50))

	model6 := `{
		"@context": {
			"definitions": {
				"address": {
					"type": "object",
					"$id" : "cc1e76d1-e341-46eb-b3ca-102ae66d82f5",
					"properties": {
						"street_address": { "type": "string" },
						"city":           { "type": "string" },
						"state":          { "type": "string" }
					},
					"required": ["street_address", "city"]
				}
			},
			"type": "object",
			"properties": {
				"billing_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" },
				"shipping_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" }
			}
		},
		"billing_address": {
			"street_address": "No. 1 Street",
			"city": "Lonton"
		},
		"shipping_address": {
			"street_address": "No. 2 Street",
			"city": "Huston",
			"state": "Texas"
		}
	}`
	schema := jsoniter.Get([]byte(model6), "@context").ToString()
	validator6, err6 := NewDataModelValidator("test5", schema, "")
	require.NotNil(t, validator6)
	require.NoError(t, err6)
	require.NoError(t, validator5.Validate(jsoniter.Get([]byte(model6))))
}
