package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/node/indexer/jobs"
	"github.com/SaoNetwork/sao-node/node/queue"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"
	"os"
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

	log.Info("building storverse views job...")
	platformId := os.Getenv("STORVERSE_PLATFORM_ID")
	if platformId == "" {
		platformId = "storverse-sao"
	}
	job := jobs.BuildStorverseViewsJob(ctx, is.ChainSvc, is.Db, platformId, log)
	is.JobsMap[job.ID] = job

	go func() {
		for {
			err := is.excute(ctx, job)
			if err != nil {
				log.Errorf("job[%s] failed due to %v", job.ID, err)
				job.Status = types.JobStatusPending
				log.Infof("Retrying job[%s]...", job.ID)
				continue
			} else {
				log.Infof("job[%s] done.", job.ID)
				continue
			}
		}
	}()

	log.Info("building metadata index job...")
	metadataJob := jobs.BuildMetadataIndexJob(ctx, is.ChainSvc, is.Db, log)
	is.JobsMap[metadataJob.ID] = metadataJob
	is.schedQueue.Push(&queue.WorkRequest{
		Job: metadataJob,
	})

	log.Info("building order sync job...")
	orderSyncJob := jobs.BuildOrderSyncJob(ctx, is.ChainSvc, is.Db, log)
	is.JobsMap[orderSyncJob.ID] = orderSyncJob
	is.schedQueue.Push(&queue.WorkRequest{
		Job: orderSyncJob,
	})

	log.Info("building node sync job...")
	nodeSyncJob := jobs.SyncNodesJob(ctx, is.ChainSvc, is.Db, log)
	is.JobsMap[nodeSyncJob.ID] = nodeSyncJob
	is.schedQueue.Push(&queue.WorkRequest{
		Job: nodeSyncJob,
	})

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
		time.Sleep(time.Second * SCHEDULE_INTERVAL)
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
