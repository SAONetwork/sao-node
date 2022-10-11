package validator

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func TestDataModelValidator(t *testing.T) {
	model1 := `{"xml": "3c726f6f742f3e"}`
	doc := jsoniter.Get([]byte(model1)).GetInterface()
	require.NotNil(t, doc)

	validator1, err1 := NewDataModelValidator("test1", `{"type": "object"}`, "")
	require.NotNil(t, validator1)
	require.NoError(t, err1)
	require.Error(t, validator1.Validate(nil))
	require.Error(t, validator1.Validate("123"))
	require.NoError(t, validator1.Validate(doc))

	validator2, err2 := NewDataModelValidator("test2", "", "")
	require.NotNil(t, validator1)
	require.NoError(t, err2)
	require.Error(t, validator2.Validate("123"))
	require.NoError(t, validator2.Validate(false))
	require.NoError(t, validator1.Validate(doc))

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
}
