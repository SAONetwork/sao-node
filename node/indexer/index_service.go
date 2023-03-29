package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"sao-node/chain"
	"sao-node/node/queue"
	"sao-node/node/repo"
	"sao-node/types"
	"sao-node/utils"
	"time"

	"github.com/ipfs/go-datastore"
	_ "github.com/mattn/go-sqlite3"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("indexer")

const (
	WINDOW_SIZE       = 20
	SCHEDULE_INTERVAL = 60
	MAX_RETRIES       = 10
)

type IndexSvcApi interface {
	AddJob(ctx context.Context, job types.Job) (string, error)
	KillJob(ctx context.Context, jobId string) error
	CheckJob(ctx context.Context) string
	ListJobs(ctx context.Context) map[string]string
}

type IndexSvc struct {
	ctx      context.Context
	chainSvc *chain.ChainSvc
	jobDs    datastore.Batching

	schedQueue *queue.RequestQueue
	locks      *utils.Maplock
	jobsMap    map[string]types.Job
	db         *sql.DB
}

func NewIndexSvc(
	ctx context.Context,
	chainSvc *chain.ChainSvc,
	repo *repo.Repo,
	jobsDs datastore.Batching,
) *IndexSvc {
	jds, err := repo.Datastore(ctx, "/jobs")
	if err != nil {
		log.Error("failed to open datastore, %v", err)
		return nil
	}

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Error("failed to open database, %v", err)
		return nil
	}

	is := &IndexSvc{
		ctx:        ctx,
		chainSvc:   chainSvc,
		jobDs:      jds,
		schedQueue: &queue.RequestQueue{},
		locks:      utils.NewMapLock(),
		jobsMap:    make(map[string]types.Job),
		db:         db,
	}

	go is.runSched(ctx)
	go is.processPendingJobs(ctx)

	return is
}

func (is *IndexSvc) runSched(ctx context.Context) {
	throttle := make(chan struct{}, WINDOW_SIZE)
	for {
		if is.schedQueue.Len() == 0 {
			time.Sleep(time.Second * SCHEDULE_INTERVAL)
			continue
		}

		len := is.schedQueue.Len()
		for i := 0; i < len; i++ {
			throttle <- struct{}{}

			go func() {
				defer func() {
					<-throttle
				}()

				sq := is.schedQueue.PopFront()
				if sq == nil || sq.Job.ID == "" {
					return
				}

				err := is.excute(ctx, &sq.Job)
				if err != nil {
					log.Warnf("failed to excute the job %v due to %v", sq.Job.ID, err)
					is.schedQueue.Push(sq)
					sq.Job.Status = types.JobStatusPending
				}
			}()
		}
	}
}

func (is *IndexSvc) processPendingJobs(ctx context.Context) {
	log.Info("process pending jobs...")

	key := datastore.NewKey("pending_jobs")
	value, err := is.jobDs.Get(ctx, key)
	if err != nil {
		log.Error("failed to obtain the pending jobs, %v", err)
	}

	var pendingJobs []queue.WorkRequest
	err = json.Unmarshal(value, &pendingJobs)
	if err != nil {
		log.Error("process pending orders error: %v", err)
	} else {
		for _, job := range pendingJobs {
			is.schedQueue.Push(&job)
		}
	}
}

func (is *IndexSvc) excute(ctx context.Context, job *types.Job) error {
	is.locks.Lock(job.ID)
	defer is.locks.Unlock(job.ID)

	is.jobsMap[job.ID] = *job
	job.Status = types.JobStatusRuning
	result, err := job.ExecFunc(ctx, job.Args)
	if err != nil {
		job.Error = err
		job.Status = types.JobStatusFailed
	} else {
		job.Result = result
		job.Status = types.JobStatusSuccessed
	}

	return err
}

func (gs *IndexSvc) Stop(ctx context.Context) error {
	log.Info("stopping index service...")

	return nil
}
