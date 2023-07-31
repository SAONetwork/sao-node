package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"sao-node/chain"
	"sao-node/types"
	"sao-node/utils"

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
		var offset uint64 = 0
		var limit uint64 = 100
		owenedMeta := make([]modeltypes.Metadata, 0)
		for {
			metaList, total, err := chainSvc.ListMeta(ctx, offset*limit, limit)
			if err != nil {
				return nil, err
			}
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

				valueArgs += fmt.Sprintf(`("%s", "%s", "%s", "%s", %d, "%s", "%s", "%s", "%s", %t, "%s", "%s", %d, %d, "%s", "%s", %d, "%s")`,
					meta.DataId, meta.Owner, meta.Alias, meta.GroupId, meta.OrderId, tags, meta.Cid, commits, meta.ExtendInfo, meta.Update, meta.Commit, meta.Rule, meta.Duration, meta.CreatedAt, readonlyDids, readwriteDids, meta.Status, strings.Trim(strings.Replace(fmt.Sprint(meta.Orders), " ", ",", -1), "[]"))

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

		return nil, nil
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build metadata index for models with specified groupIds",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
