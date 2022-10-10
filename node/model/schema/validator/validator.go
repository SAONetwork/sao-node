package validator

import (
	"sao-storage-node/node/model/rule_engine"
	"strings"

	jsoniter "github.com/json-iterator/go"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/xerrors"
)

const Draft7_Url = "https://json-schema.org/draft-07/schema"
const Prefix_Context = "Context_"
const Prefix_Rule = "Rule_"

type (
	Validator struct {
		name string
		sch  *jsonschema.Schema
		svc  *rule_engine.RuleEngineSvc
	}

	Result struct {
		IsValid bool
		Reason  string
	}
)

func NewDataModelValidator(dmName string, dmSchema string, dmRule string) (*Validator, error) {
	url := dmName + ".json"
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	if dmSchema != "" {
		if err := compiler.AddResource(url, strings.NewReader(dmSchema)); err != nil {
			return nil, err
		}
	} else {
		url = Draft7_Url
	}

	schema, err := compiler.Compile(url)
	if err != nil {
		return nil, err
	}

	if dmRule != "" {
		ruleEngineSvc := rule_engine.NewRuleEngineSvc()
		err = ruleEngineSvc.AddRule(Prefix_Rule+dmName, []byte(dmRule))
		if err != nil {
			return nil, err
		}

		return &Validator{
			name: dmName,
			sch:  schema,
			svc:  ruleEngineSvc,
		}, nil
	}

	return &Validator{
		name: dmName,
		sch:  schema,
		svc:  nil,
	}, nil
}

func (v *Validator) ValidateWithRef(dmContent interface{}, refContents map[string]interface{}) error {
	model, err := jsoniter.Marshal(dmContent)
	if err != nil {
		return xerrors.Errorf(err.Error())
	}
	doc := jsoniter.Get(model).GetInterface()

	err = v.sch.Validate(doc)
	if err == nil {
		if v.svc == nil {
			return nil
		} else {
			v.svc.Reset(Prefix_Context + v.name)

			err = v.svc.AddFact(Prefix_Context+v.name, v.name, dmContent)
			if err != nil {
				return xerrors.Errorf(err.Error())
			}

			for name, refModel := range refContents {
				err = v.svc.AddFact(Prefix_Context+v.name, name, refModel)
				if err != nil {
					return xerrors.Errorf(err.Error())
				}
			}

			result := &Result{
				IsValid: true,
				Reason:  "",
			}

			err = v.svc.AddFact(Prefix_Context+v.name, "Result", result)
			if err != nil {
				return xerrors.Errorf(err.Error())
			}

			err = v.svc.Execute(Prefix_Rule+v.name, Prefix_Context+v.name)
			if err == nil {
				if result.IsValid {
					return nil
				} else {
					return xerrors.Errorf("failed to pass the rule check due to " + result.Reason)
				}
			} else {
				return xerrors.Errorf(err.Error())
			}
		}
	}

	if ve, ok := err.(*jsonschema.ValidationError); ok {
		if len(ve.Causes) == 1 {
			field := ve.Causes[0].InstanceLocation
			if len(field) > 0 && field[0] == '/' {
				field = field[1:]
			}

			return xerrors.Errorf("validation failed, invalid field '%s' due to '%s'", field, ve.Causes[0].Message)
		}
	}

	return xerrors.Errorf(err.Error())
}

func (v *Validator) Validate(dmContent interface{}) error {
	return v.ValidateWithRef(dmContent, nil)
}
