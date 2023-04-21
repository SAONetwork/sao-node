package types

import (
	"context"
)

type ExecutionFunc func(ctx context.Context, args []interface{}) (interface{}, error)

const (
	JobStatusPending   = "Pending"
	JobStatusRuning    = "Runing"
	JobStatusSuccessed = "Successed"
	JobStatusFailed    = "Failed"
	JobStatusKilled    = "Killed"
)

type Job struct {
	ID          string
	Description string
	Status      string
	ExecFunc    ExecutionFunc
	Args        []interface{}
	Result      interface{}
	Error       error
}

func (j Job) Execute(ctx context.Context) (interface{}, error) {
	value, err := j.ExecFunc(ctx, j.Args)
	if err != nil {
		return nil, err
	}

	return value, nil
}
