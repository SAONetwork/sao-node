package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"
	logging "github.com/ipfs/go-log/v2"
)

//go:embed sqls/create_orders_table.sql
var createOrdersDBSQL string

func BuildOrderSyncJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, log *logging.ZapEventLogger) *types.Job {
	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		// initialize the orders database tables
		log.Info("creating orders tables...")
		if _, err := db.ExecContext(ctx, createOrdersDBSQL); err != nil {
			log.Errorf("failed to create tables: %w", err)
			return nil, err
		}
		log.Info("creating orders tables done.")

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
				qry := "SELECT COUNT(*) FROM ORDERS WHERE id=?"
				row := db.QueryRowContext(ctx, qry, order.Id)
				var count int
				err := row.Scan(&count)
				if err != nil {
					return nil, err
				}

				if count == 0 {
					stmt := `INSERT INTO ORDERS 
					(creator, owner, id, provider, cid, duration, status, replica, amount, size, operation, createdAt, timeout, dataId, commitId, unitPrice)
					VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

					_, err := db.ExecContext(ctx, stmt,
						order.Creator, order.Owner, order.Id, order.Provider, order.Cid, order.Duration,
						order.Status, order.Replica, order.Amount.Amount.String(), order.Size_, order.Operation, order.CreatedAt,
						order.Timeout, order.DataId, order.Commit, order.UnitPrice.Amount.String())

					if err != nil {
						return nil, err
					}
					log.Infof("insert order %s", order.Id)
				}
			}
		}

		return nil, nil
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build order sync for models",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
