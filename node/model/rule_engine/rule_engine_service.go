package rule_engine

import (
	"sao-node/types"
	"sync"

	"github.com/hyperjumptech/grule-rule-engine/ast"
	"github.com/hyperjumptech/grule-rule-engine/builder"
	"github.com/hyperjumptech/grule-rule-engine/engine"
	"github.com/hyperjumptech/grule-rule-engine/pkg"
)

type RuleEngineSvc struct {
	knowledgeLibrary *ast.KnowledgeLibrary
	rulesMap         map[string]*ast.KnowledgeBase
	dataCtxMap       map[string]ast.IDataContext
}

var (
	ruleEngineSvc *RuleEngineSvc
	once          sync.Once
)

func NewRuleEngineSvc() *RuleEngineSvc {
	once.Do(func() {
		ruleEngineSvc = &RuleEngineSvc{
			knowledgeLibrary: ast.NewKnowledgeLibrary(),
			rulesMap:         make(map[string]*ast.KnowledgeBase),
			dataCtxMap:       make(map[string]ast.IDataContext),
		}
	})
	return ruleEngineSvc
}

func (svc *RuleEngineSvc) AddRule(name string, jsonData []byte) error {
	rule, err := pkg.ParseJSONRule(jsonData)
	if err != nil {
		return err
	}

	ruleBuilder := builder.NewRuleBuilder(svc.knowledgeLibrary)
	err = ruleBuilder.BuildRuleFromResource(name, "v0", pkg.NewBytesResource([]byte(rule)))
	if err != nil {
		return err
	}
	knowledgeBase := svc.knowledgeLibrary.NewKnowledgeBaseInstance(name, "v0")
	svc.rulesMap[name] = knowledgeBase

	return nil
}

func (svc *RuleEngineSvc) AddFact(dataCtxName string, factName string, fact interface{}) error {
	dataCtx := svc.dataCtxMap[dataCtxName]
	if dataCtx == nil {
		dataCtx = ast.NewDataContext()
		svc.dataCtxMap[dataCtxName] = dataCtx
	}
	err := dataCtx.Add(factName, fact)
	if err != nil {
		return err
	}

	return nil
}

func (svc *RuleEngineSvc) Reset(dataCtxName string) {
	dataCtx := svc.dataCtxMap[dataCtxName]
	if dataCtx == nil {
		dataCtx = ast.NewDataContext()
		svc.dataCtxMap[dataCtxName] = dataCtx
	}
	dataCtx.Reset()
}

func (svc *RuleEngineSvc) Clear(dataCtxName string) {
	dataCtx := svc.dataCtxMap[dataCtxName]
	if dataCtx != nil {
		dataCtx.Reset()
		svc.dataCtxMap[dataCtxName] = nil
	}
}

func (svc *RuleEngineSvc) Execute(ruleName string, dataCtxName string) error {
	knowledgeBase := svc.rulesMap[ruleName]
	if knowledgeBase == nil {
		return types.Wrapf(types.ErrRuleExcuteFaild, "the rule [%s] not found", ruleName)
	}

	dataCtx := svc.dataCtxMap[dataCtxName]
	if dataCtx == nil {
		return types.Wrapf(types.ErrRuleExcuteFaild, "the datacontext [%s] not found", dataCtxName)
	}

	ruleEngine := engine.NewGruleEngine()
	err := ruleEngine.Execute(dataCtx, knowledgeBase)
	if err != nil {
		return err
	} else {
		return nil
	}
}
