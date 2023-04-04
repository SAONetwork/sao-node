package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sao-node/chain"
	"sao-node/node/queue"
	"sao-node/types"
	"sao-node/utils"
	"strings"
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
	ChainSvc *chain.ChainSvc
	jobDs    datastore.Batching

	schedQueue *queue.RequestQueue
	locks      *utils.Maplock
	JobsMap    map[string]*types.Job
	Db         *sql.DB
}

func NewIndexSvc(
	ctx context.Context,
	chainSvc *chain.ChainSvc,
	jobsDs datastore.Batching,
	dbPath string,
) *IndexSvc {
	db, err := sql.Open("sqlite3", dbPath+"/indexer.db")
	if err != nil {
		log.Error("failed to open database, %v", err)
		return nil
	}

	is := &IndexSvc{
		ctx:        ctx,
		ChainSvc:   chainSvc,
		jobDs:      jobsDs,
		schedQueue: &queue.RequestQueue{},
		locks:      utils.NewMapLock(),
		JobsMap:    make(map[string]*types.Job),
		Db:         db,
	}

	go is.runSched(ctx)
	go is.processPendingJobs(ctx)

	// two examples
	// examples1: create a job to collect the metadata created on dapp whose platform id is 30293f0f-3e0f-4b3c-aff1-890a2fdf063b
	// job1 := jobs.BuildMetadataIndexJob(ctx, is.ChainSvc, is.Db, "30293f0f-3e0f-4b3c-aff1-890a2fdf063b")
	// is.JobsMap[job1.ID] = job1
	// is.schedQueue.Push(&queue.WorkRequest{
	// 	Job: job1,
	// })

	// examples2: create a job to collect the shards which assigned to sp 30293f0f-3e0f-4b3c-aff1-890a2fdf063b
	// job2 := jobs.BuildSpShardIndexJob(ctx, is.ChainSvc, is.Db, "sao1ek2nmuzjc479kz78qun00v30j8whpt52vcarme")
	// is.JobsMap[job2.ID] = job2
	// is.schedQueue.Push(&queue.WorkRequest{
	// 	Job: job2,
	// })

	return is
}

func (is *IndexSvc) runSched(ctx context.Context) {
	throttle := make(chan struct{}, WINDOW_SIZE)
	for {
		if is.schedQueue.Len() == 0 {
			log.Info("no job found.")
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

				log.Infof("job[%s] loaded.", sq.Job.ID)

				err := is.excute(ctx, sq.Job)
				log.Infof("job[%s] running...", sq.Job.ID)

				if err != nil {
					log.Errorf("job[%s] failed due to %v", sq.Job.ID, err)
					is.schedQueue.Push(sq)
					sq.Job.Status = types.JobStatusPending
				} else {
					log.Infof("job[%s] done.", sq.Job.ID)
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
		if strings.Contains(fmt.Sprintf("%v", err), "key not found") {
			log.Infof("no pending jobs found")
		} else {
			log.Errorf("failed to obtain the pending jobs, %v", err)
		}
	} else {
		var pendingJobs []queue.WorkRequest
		err = json.Unmarshal(value, &pendingJobs)
		if err != nil {
			log.Errorf("process pending jobs error: %v", err)
		} else {
			for _, job := range pendingJobs {
				is.schedQueue.Push(&job)
			}
		}
	}
}

func (is *IndexSvc) excute(ctx context.Context, job *types.Job) error {
	is.locks.Lock(job.ID)
	defer is.locks.Unlock(job.ID)

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
