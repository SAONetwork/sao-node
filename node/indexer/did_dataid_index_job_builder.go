package indexer

import (
	"context"
	"fmt"
	"sao-node/types"
	"sao-node/utils"
	"strings"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	"github.com/ipfs/go-datastore"
)

func (is *IndexSvc) BuildDataIdIndexJob(ctx context.Context, platFormId string) types.Job {
	execFn := func(ctx context.Context, args []interface{}) (interface{}, error) {
		key := datastore.NewKey("did_dataid_index_job_last_loaded_id")
		lastLoadId := ""
		value, err := is.jobDs.Get(ctx, key)
		if err != nil {
			lastLoadId = string(value)
		}

		var offset uint64 = 0
		var limit uint64 = 100
		owenedMeta := make([]modeltypes.Metadata, 0)
		for {
			metaList, total, err := is.chainSvc.ListMeta(ctx, offset, limit)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				break
			}

			for _, meta := range metaList {
				if lastLoadId != "" && lastLoadId == meta.Commit {
					break
				} else {
					if meta.GroupId == platFormId {
						owenedMeta = append(owenedMeta, meta)
					}
				}
			}
		}

		if len(owenedMeta) > 0 {
			err := is.jobDs.Put(ctx, key, []byte(lastLoadId))
			if err != nil {
				return nil, err
			}

			var valueStrings []string
			if len(owenedMeta) > 500 {
				valueStrings = make([]string, 0, 500)
			} else {
				valueStrings = make([]string, 0, len(owenedMeta))
			}
			valueArgs := make([]interface{}, 0, len(owenedMeta)*3)
			for index, meta := range owenedMeta {
				valueArgs := fmt.Sprintf("%s (%s, %s, %s, %s, %s, v%d, %d, %d, %d, %s, %s)",
					valueArgs, meta.Owner, meta.DataId, meta.Alias, meta.GroupId, meta.Commit, len(meta.Commits), meta.Size(), meta.CreatedAt, meta.Duration, meta.ReadonlyDids, meta.ReadwriteDids)

				if index%500 == 0 {
					stmt := fmt.Sprintf("INSERT INTO METADATA (OWNER, DATAID, NAME, PLAT, COMMITID, VERSION, SIZE, EXPIRATION, READER, WRITER) VALUES %s",
						valueArgs)
					_, err := is.db.Exec(stmt)
					if err != nil {
						return nil, err
					}
				} else {
					valueArgs += ","
				}
			}
			stmt := fmt.Sprintf("INSERT INTO METADATA (OWNER, DATAID, NAME, PLAT, COMMITID, VERSION, SIZE, EXPIRATION, READER, WRITER) VALUES %s",
				strings.Join(valueStrings, ","))
			_, err = is.db.Exec(stmt, valueArgs...)
			if err != nil {
				return nil, err
			}
			log.Infof("batch done, %d metadata records loaded.", len(owenedMeta))
		}

		return nil, nil
	}

	return types.Job{
		ID:       utils.GenerateDataId("job-id"),
		Status:   types.JobStatusPending,
		ExecFunc: execFn,
		Args:     make([]interface{}, 0),
	}
}
