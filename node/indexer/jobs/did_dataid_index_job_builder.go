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

var log = logging.Logger("indexer-jobs")

//go:embed sqls/create_meta_data_table.sql
var createMetaDBSQL string

func BuildMetadataIndexJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, platFormIds string) types.Job {
	// initialize the metadata database tables
	log.Info("creating metadata tables...")
	if _, err := db.ExecContext(ctx, createMetaDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating metadata tables done.")

	execFn := func(ctx context.Context, args []interface{}) (interface{}, error) {
		var offset uint64 = 0
		var limit uint64 = 100
		owenedMeta := make([]modeltypes.Metadata, 0)
		for {
			metaList, total, err := chainSvc.ListMeta(ctx, offset, limit)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				break
			}

			for _, meta := range metaList {
				qry := "SELECT COUNT(*) FROM METADATA WHERE COMMITID=?"
				row := db.QueryRowContext(ctx, qry, meta.Commit)
				var count int
				err := row.Scan(&count)
				if err != nil {
					return nil, err
				}

				if count > 0 {
					continue
				} else {
					if strings.Contains(platFormIds, meta.GroupId) {
						owenedMeta = append(owenedMeta, meta)
					}
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

				valueArgs = fmt.Sprintf(`%s("%s", "%s", "%s", "%s", "%s", "v%d", %d, %d, "%s", "%s")`,
					valueArgs, meta.Owner, meta.DataId, meta.Alias, meta.GroupId, meta.Commit, len(meta.Commits), meta.Size(), meta.CreatedAt+meta.Duration, meta.ReadonlyDids, meta.ReadwriteDids)

				if index > 0 && index%500 == 0 {
					log.Infof("sub batch prepare, 500 metadata records to be saved.")
					stmt := fmt.Sprintf("INSERT INTO METADATA (DID, DATAID, ALIAS, PLAT, COMMITID, VER, SIZE, EXPIRATION, READER, WRITER) VALUES %s",
						valueArgs)
					_, err := db.Exec(stmt)
					if err != nil {
						return nil, err
					}
					valueArgs = ""
					log.Infof("sub batch done, %d metadata records saved.", len(owenedMeta))
				}
			}
			if len(valueArgs) > 0 {
				stmt := fmt.Sprintf("INSERT INTO METADATA (DID, DATAID, ALIAS, PLAT, COMMITID, VER, SIZE, EXPIRATION, READER, WRITER) VALUES %s",
					valueArgs)
				_, err := db.Exec(stmt)
				if err != nil {
					return nil, err
				}
				log.Infof("batch done, %d metadata records saved.", len(owenedMeta))
			}
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
