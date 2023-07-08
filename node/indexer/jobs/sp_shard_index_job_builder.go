package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"
)

//go:embed sqls/create_sp_shard_table.sql
var createSpShardDBSQL string

type Shard struct {
	ShardId uint64
	OrderId uint64
	Sp      string
	Cid     string
}

func BuildSpShardIndexJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, providers string) *types.Job {
	// initialize the sp shard database tables
	log.Info("creating sp shard tables...")
	if _, err := db.ExecContext(ctx, createSpShardDBSQL); err != nil {
		log.Error("failed to create tables: ", err)
	}
	log.Info("creating sp shard tables done.")

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		var offset uint64 = 0
		var limit uint64 = 100
		shards := make([]Shard, 0)
		for {
			shardList, total, err := chainSvc.ListShards(ctx, offset, limit)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				break
			}

			for _, shard := range shardList {
				qry := "SELECT COUNT(*) FROM SP_SHARD WHERE ORDERID=? AND SHARDID=?"
				row := db.QueryRowContext(ctx, qry, shard.OrderId, shard.Id)
				var count int
				err := row.Scan(&count)
				if err != nil {
					return nil, err
				}

				if count > 0 {
					continue
				} else {
					if strings.Contains(providers, shard.Sp) {
						shards = append(shards, Shard{
							ShardId: shard.Id,
							OrderId: shard.OrderId,
							Sp:      shard.Sp,
							Cid:     shard.Cid,
						})
					}
				}
			}

		}

		if len(shards) > 0 {
			valueArgs := ""
			log.Infof("batch prepare, %d sp shard records to be saved.", len(shards))
			for index, shard := range shards {
				if valueArgs != "" {
					valueArgs += ", "
				}

				valueArgs = fmt.Sprintf(`%s(%d, %d, "%s", "%s")`,
					valueArgs, shard.ShardId, shard.OrderId, shard.Sp, shard.Cid)

				if index > 0 && index%500 == 0 {
					log.Infof("sub batch prepare, 500 sp shard records to be saved.")
					stmt := fmt.Sprintf("INSERT INTO SP_SHARD (SHARDID, ORDERID, SP, CID) VALUES %s",
						valueArgs)
					_, err := db.Exec(stmt)
					if err != nil {
						return nil, err
					}
					valueArgs = ""
					log.Infof("sub batch done, %d sp shard records saved.", len(shards))
				}
			}
			if len(valueArgs) > 0 {
				stmt := fmt.Sprintf("INSERT INTO SP_SHARD (SHARDID, ORDERID, SP, CID) VALUES %s",
					valueArgs)
				_, err := db.Exec(stmt)
				if err != nil {
					return nil, err
				}
				log.Infof("batch done, %d sp shard records saved.", len(shards))
			}
		}

		return nil, nil
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build sp shard index for order with specified sp address",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
