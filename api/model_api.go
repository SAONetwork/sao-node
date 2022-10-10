package api

import (
	"sao-storage-node/node/model"
)

type ModelApi interface {
	// MethodGroup: Data Model

	Load(account string, alias string) (model.Model, error)                                //perm:read
	Create(account string, alias string, context string, rule string) (model.Model, error) //perm:write
	Update(account string, alias string, patch string, rule string) (model.Model, error)   //perm:write
}
