package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	logging "github.com/ipfs/go-log/v2"
)

//go:embed sqls/create_metadata_table.sql
var createMetadataDBSQL string

func BuildMetadataIndexJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, log *logging.ZapEventLogger) *types.Job {
	// initialize the metadata database tables
	log.Info("creating metadata tables...")
	if _, err := db.ExecContext(ctx, createMetadataDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating metadata tables done.")

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		// Drop the temporary table if it exists
		_, err := db.Exec("DROP TABLE IF EXISTS TempMetadata")
		if err != nil {
			log.Errorf("failed to drop temporary table: %w", err)
		}
		// Create a temporary table
		_, err = db.Exec("CREATE TABLE TempMetadata (dataId VARCHAR(255))")
		if err != nil {
			log.Errorf("failed to create temporary table: %w", err)
			return nil, err
		}
		log.Info("created temporary table")

		var offset uint64 = 0
		var limit uint64 = 200
		owenedMeta := make([]modeltypes.Metadata, 0)
		for {
			metaList, total, err := chainSvc.ListMeta(ctx, offset*limit, limit)
			if err != nil {
				log.Error("Failed to read metadata list")
				return nil, err
			}

			if len(metaList) == 0 {
				break
			}

			var dataIds []string
			var placeholders []string

			for _, meta := range metaList {
				dataIds = append(dataIds, meta.DataId)
				placeholders = append(placeholders, "(?)")
			}

			// Capture the start time before the insert
			startTime := time.Now()
			// Insert the current batch into TempMetadata
			stmt := fmt.Sprintf("INSERT INTO TempMetadata (dataId) VALUES %s", strings.Join(placeholders, ","))
			_, err = db.Exec(stmt, ConvertToInterfaceSlice(dataIds)...)
			if err != nil {
				log.Errorf("failed to insert into temporary table: %w", err)
				return nil, err
			}

			// Capture the end time after the insert
			endTime := time.Now()

			// Calculate and log the duration
			duration := endTime.Sub(startTime)
			log.Infof("Time spent on insert: %v", duration)

			if offset*limit <= total {
				offset++
			} else {
				break
			}

			for _, meta := range metaList {
				qry := "SELECT COUNT(*) FROM METADATA WHERE DATAID=? AND `commitId`=?"
				row := db.QueryRowContext(ctx, qry, meta.DataId, meta.Commit)
				var count int
				err := row.Scan(&count)
				if err != nil {
					return nil, err
				}

				if count > 0 {
					continue
				} else {
					qry := "DELETE FROM METADATA WHERE DATAID=? AND `commitId`<>?"
					_, err = db.ExecContext(ctx, qry, meta.DataId, meta.Commit)
					if err != nil {
						return nil, err
					}
					owenedMeta = append(owenedMeta, meta)
				}

			}
		}
		log.Info("read metadata done")

		// Delete from METADATA table where dataId is not in the temporary table
		query := "DELETE FROM METADATA WHERE dataId NOT IN (SELECT dataId FROM TempMetadata)"
		result, err := db.Exec(query)
		if err != nil {
			log.Errorf("failed to delete metadata: %w", err)
			return nil, err
		}

		rowsDeleted, err := result.RowsAffected()
		if err != nil {
			log.Errorf("failed to retrieve the count of deleted rows: %w", err)
			return nil, err
		}
		log.Infof("%d rows deleted from METADATA table", rowsDeleted)

		if len(owenedMeta) > 0 {
			valueArgs := ""
			log.Infof("batch prepare, %d metadata records to be saved.", len(owenedMeta))
			for index, meta := range owenedMeta {
				if valueArgs != "" {
					valueArgs += ", "
				}

				tags := strings.Join(meta.Tags, ",")
				commits := strings.Join(meta.Commits, ",")
				readonlyDids := strings.Join(meta.ReadonlyDids, ",")
				readwriteDids := strings.Join(meta.ReadwriteDids, ",")

				chainSvc.ListNodes(ctx)

				resBlock, err := chainSvc.GetBlock(ctx, int64(meta.CreatedAt))
				if err != nil {
					log.Errorf("failed to get block at height %d for metadata %s: %w", meta.CreatedAt, meta.DataId, err)
					return nil, err
				}

				valueArgs += fmt.Sprintf(`("%s", "%s", "%s", "%s", %d, "%s", "%s", "%s", "%s", %t, "%s", "%s", %d, %d, "%s", "%s", %d, "%s")`,
					meta.DataId, meta.Owner, meta.Alias, meta.GroupId, meta.OrderId, tags, meta.Cid, commits, meta.ExtendInfo, meta.Update, meta.Commit, meta.Rule, meta.Duration, resBlock.Block.Header.Time.Unix(), readonlyDids, readwriteDids, meta.Status, strings.Trim(strings.Replace(fmt.Sprint(meta.Orders), " ", ",", -1), "[]"))

				if index > 0 && index%500 == 0 {
					log.Infof("sub batch prepare, 500 metadata records to be saved.")
					stmt := fmt.Sprintf("INSERT INTO METADATA (dataId, owner, alias, groupId, orderId, tags, cid, `commits`, extendInfo, `updateAt`, `commitId`, rule, duration, createdAt, readonlyDids, readwriteDids, status, orders) VALUES %s",
						valueArgs)
					_, err := db.Exec(stmt)
					if err != nil {
						log.Errorf("failed to save metadata: %w", err)
						return nil, err
					}
					valueArgs = ""
					log.Infof("sub batch done, %d metadata records saved.", len(owenedMeta))
				}
			}
			if len(valueArgs) > 0 {
				stmt := fmt.Sprintf("INSERT INTO METADATA (dataId, owner, alias, groupId, orderId, tags, cid, commits, extendInfo, `updateAt`, `commitId`, rule, duration, createdAt, readonlyDids, readwriteDids, status, orders) VALUES %s",
					valueArgs)
				_, err := db.Exec(stmt)
				if err != nil {
					log.Errorf("failed to save metadata: %w", err)
					return nil, err
				}
				log.Infof("batch done, %d metadata records saved.", len(owenedMeta))
			}
		}

		return nil, errors.New("we will trigger next sync")
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build metadata index for models with specified groupIds",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}

func ConvertToInterfaceSlice(strSlice []string) []interface{} {
	ifaceSlice := make([]interface{}, len(strSlice))
	for i, v := range strSlice {
		ifaceSlice[i] = v
	}
	return ifaceSlice
}
