package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
	"time"

	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"

	logging "github.com/ipfs/go-log/v2"
)

//go:embed sqls/create_node_table.sql
var createNodesTableSQL string

func SyncNodesJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, log *logging.ZapEventLogger) *types.Job {
	log.Info("creating nodes tables...")
	if _, err := db.ExecContext(ctx, createNodesTableSQL); err != nil {
		log.Errorf("failed to create nodes table: %w", err)
	}
	log.Info("creating nodes tables done.")

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		nodes, err := chainSvc.ListNodes(ctx)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			// Check if the node with the specific Creator and Peer already exists
			qry := "SELECT COUNT(*) FROM NODE WHERE Creator=?"
			row := db.QueryRowContext(ctx, qry, node.Creator)
			var count int
			err := row.Scan(&count)
			if err != nil {
				return nil, err
			}

			// Extract specific flags from the Status bitmap
			isGateway := (node.Status & 2) != 0
			isSP := (node.Status & 4) != 0
			isIndexer := (node.Status & 16) != 0

			// Convert LastAliveHeight to Unix timestamp
			resBlock, err := chainSvc.GetBlock(ctx, int64(node.LastAliveHeight))
			if err != nil {
				log.Errorf("failed to get block at height %d for node %s: %w", node.LastAliveHeight, node.Creator, err)
				return nil, err
			}

			// Check if the LastAliveHeight is within the last 24 hours
			isAlive := time.Now().Unix()-resBlock.Block.Header.Time.Unix() <= 24*60*60

			txAddresses := strings.Join(node.TxAddresses, ",")

			// If node info doesn't exist in the database, insert it
			if count == 0 {
				query := `INSERT INTO NODE (Creator, Peer, Reputation, Status, LastAliveHeight, TxAddresses, Role, Validator, IsGateway, IsSP, IsIndexer, IsAlive, LastAliveTime, IPAddress, Name)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

				_, err := db.ExecContext(ctx, query, node.Creator, node.Peer, node.Reputation, node.Status, node.LastAliveHeight, txAddresses, node.Role, node.Validator, isGateway, isSP, isIndexer, isAlive, resBlock.Block.Header.Time.Unix(), "", "")
				if err != nil {
					log.Errorf("failed to insert node data for %s: %w", node.Creator, err)
					return nil, err
				}
			}
		}
		log.Infof("Sync done, %d nodes records updated.", len(nodes))
		return nil, nil
	}

	return &types.Job{
		ID:          utils.GenerateDataId("node-sync-job-id"),
		Description: "sync node data from the chain",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
