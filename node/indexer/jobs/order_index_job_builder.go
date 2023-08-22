package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"
	logging "github.com/ipfs/go-log/v2"
	"strconv"
	"strings"
)

//go:embed sqls/create_orders_table.sql
var createOrdersDBSQL string

func BuildOrderSyncJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, log *logging.ZapEventLogger) *types.Job {
	// initialize the orders database tables
	log.Info("creating orders tables...")
	if _, err := db.ExecContext(ctx, createOrdersDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating orders tables done.")

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		var offset uint64 = 0
		var limit uint64 = 100

		for {
			orderList, total, err := chainSvc.GetAllOrders(ctx, offset*limit, limit)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				break
			}

			for _, order := range orderList {
				qry := "SELECT status FROM ORDERS WHERE id=?"
				row := db.QueryRowContext(ctx, qry, order.Id)
				var existingStatus int32
				err := row.Scan(&existingStatus)

				if err != nil && err != sql.ErrNoRows {
					return nil, err
				}

				if err == nil && existingStatus != order.Status {
					// The order exists, and the status has changed, so delete the existing record
					qry := "DELETE FROM ORDERS WHERE id=?"
					_, err = db.ExecContext(ctx, qry, order.Id)
					if err != nil {
						return nil, err
					}
				}

				resBlock, err := chainSvc.GetBlock(ctx, int64(order.CreatedAt))
				if err != nil {
					log.Errorf("failed to get block at height %d for order %s: %w", order.CreatedAt, order.Id, err)
					return nil, err
				}

				shards := ""
				for _, shard := range order.Shards {
					shards += strconv.FormatUint(shard, 10) + ","
				}
				shards = strings.TrimSuffix(shards, ",")

				stmt := `INSERT INTO ORDERS 
					(creator, owner, id, provider, cid, duration, status, replica, shards, amount, size, operation, createdAt, timeout, dataId, commitId, unitPrice)
					VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

				_, err = db.ExecContext(ctx, stmt,
					order.Creator, order.Owner, order.Id, order.Provider, order.Cid, order.Duration,
					order.Status, order.Replica, shards, order.Amount.Amount.String(), order.Size_, order.Operation, resBlock.Block.Header.Time.Unix(),
					order.Timeout, order.DataId, order.Commit, order.UnitPrice.Amount.String())

				if err != nil {
					return nil, err
				}
				log.Infof("insert order %s", order.Id)
			}
		}

		return nil, errors.New("we will trigger next sync")
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build order sync for models",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
