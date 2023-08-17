package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"net"
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
			// Check if the node with the specific Creator already exists
			qry := "SELECT LastAliveHeight FROM NODE WHERE Creator=?"
			row := db.QueryRowContext(ctx, qry, node.Creator)
			var existingLastAliveHeight int64
			err := row.Scan(&existingLastAliveHeight)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Errorf("failed to query node data for %s: %w", node.Creator, err)
				continue
			}

			// If node info does not exist or LastAliveHeight has changed, insert or update
			if errors.Is(err, sql.ErrNoRows) || existingLastAliveHeight != node.LastAliveHeight {
				if existingLastAliveHeight != node.LastAliveHeight && !errors.Is(err, sql.ErrNoRows) {
					deleteQuery := "DELETE FROM NODE WHERE Creator=?"
					_, err := db.ExecContext(ctx, deleteQuery, node.Creator)
					if err != nil {
						log.Errorf("failed to delete node data for %s with different LastAliveHeight: %w", node.Creator, err)
						return nil, err
					}
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

				// Extracting IPAddress from node.Peer
				peerAddresses := strings.Split(node.Peer, ",")
				var ipAddress string
				for _, address := range peerAddresses {
					// Split by '/'
					parts := strings.Split(address, "/")
					// If the part contains an IP, then check if it's non-internal
					if len(parts) > 2 {
						ip := net.ParseIP(parts[2])
						// Check if the IP is valid and is not an internal IP
						if ip != nil && !ip.IsLoopback() && !ip.IsPrivate() {
							ipAddress = ip.String()
							break
						}
					}
				}

				// Insert the new node data
				query := `INSERT INTO NODE (Creator, Peer, Reputation, Status, LastAliveHeight, TxAddresses, Role, Validator, IsGateway, IsSP, IsIndexer, IsAlive, LastAliveTime, IPAddress, Name)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

				_, err = db.ExecContext(ctx, query, node.Creator, node.Peer, node.Reputation, node.Status, node.LastAliveHeight, txAddresses, node.Role, node.Validator, isGateway, isSP, isIndexer, isAlive, resBlock.Block.Header.Time.Unix(), ipAddress, "")
				if err != nil {
					log.Errorf("failed to insert node data for %s: %w", node.Creator, err)
					return nil, err
				}
			}
		}
		log.Infof("Sync done, %d nodes records updated.", len(nodes))
		return nil, errors.New("we will trigger next sync")
	}

	return &types.Job{
		ID:          utils.GenerateDataId("node-sync-job-id"),
		Description: "sync node data from the chain",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
