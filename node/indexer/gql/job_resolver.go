package gql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
)

type job struct {
	JobId       string
	Description string
	Status      string
}

type jobList struct {
	TotalCount int32
	Jobs       []*job
	More       bool
}

// query: job(id) Job
func (r *resolver) Job(ctx context.Context, args struct{ ID graphql.ID }) (*job, error) {
	var jobId uuid.UUID
	err := jobId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	j := r.indexSvc.JobsMap[jobId.String()]
	if j == nil {
		return nil, fmt.Errorf("no job[%s] found", jobId)
	} else {
		return &job{
			JobId:       jobId.String(),
			Description: j.Description,
			Status:      j.Status,
		}, nil
	}
}

// query: jobs(cursor, offset, limit) JobList
func (r *resolver) Jobs(ctx context.Context, args struct{ Query graphql.NullString }) (*jobList, error) {
	jobs := make([]*job, 0)
	for jobId, j := range r.indexSvc.JobsMap {
		jobs = append(jobs, &job{
			JobId:       jobId,
			Description: j.Description,
			Status:      j.Status,
		})
	}

	return &jobList{
		TotalCount: int32(len(jobs)),
		Jobs:       jobs,
		More:       false,
	}, nil
}

func (m *job) ID() graphql.ID {
	return graphql.ID(m.JobId)
}
