package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	did "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	logging "github.com/ipfs/go-log/v2"
	"os"
	"reflect"
	"sao-node/api"
	apiclient "sao-node/api/client"
	apitypes "sao-node/api/types"
	"sao-node/chain"
	"sao-node/node/indexer/jobs/storverse/model"
	"sao-node/types"
	"sao-node/utils"
	"strings"
	"time"
)

//var log = logging.Logger("indexer-jobs")

//go:embed sqls/create_user_profile_table.sql
var createUserProfileDBSQL string

//go:embed sqls/create_verse_table.sql
var createVerseDBSQL string

//go:embed sqls/create_file_info_table.sql
var createFileInfoDBSQL string

//go:embed sqls/create_following_table.sql
var createUserFollowingDBSQL string

type BatchInserter interface {
	InsertValues() string
}

func BuildStorverseViewsJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, platFormIds string, log *logging.ZapEventLogger) *types.Job {
	// initialize the tables
	InitializeStorverseTables(ctx, log, db)

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		// Get the gateway api
		gwAddress := os.Getenv("SAO_GATEWAY_API")
		if gwAddress == "" {
			gwAddress = "http://127.0.0.1:5151/rpc/v0"
		}
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gwAddress, "DEFAULT_TOKEN")
		if err != nil {
			log.Errorf("failed to get gateway api: %w", err)
			return nil, err
		}
		defer closer()

		keyName := os.Getenv("STORVERSE_KEY_NAME")
		keyringHome := os.Getenv("SAO_KEYRING_HOME")
		log.Infof("keyName: %s, keyringHome: %s", keyName, keyringHome)
		if keyName == "" || keyringHome == "" {
			log.Error("keyName or keyringHome is empty, please set the env variable")
			return nil, err
		}
		didManager, _, err := GetDidManager(ctx, keyName, keyringHome)
		if err != nil {
			log.Errorf("failed to get did manager: %w", err)
			return nil, err
		}

		gatewayAddress, err := gatewayApi.GetNodeAddress(ctx)
		if err != nil {
			log.Errorf("failed to get gateway address: %w", err)
			return nil, err
		}

		var offset uint64 = 0
		var limit uint64 = 500
		// Create slice to store the user data
		//var usersToUpdate []storverse.UserProfile
		var usersToCreate []storverse.UserProfile
		//var versesToUpdate []storverse.Verse
		var versesToCreate []storverse.Verse
		//var fileInfosToUpdate []storverse.FileInfo
		var fileInfosToCreate []storverse.FileInfo
		//var userFollowingToUpdate []storverse.UserFollowing
		var userFollowingToCreate []storverse.UserFollowing
		commitIds := make(map[string]bool)
		for {
			metaList, total, err := chainSvc.ListMeta(ctx, offset, limit)
			log.Infof("offset: %d, limit: %d, total: %d", offset, limit, total)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				// Convert []UserProfile to []BatchInserter
				userProfileBatchInserters := make([]BatchInserter, len(usersToCreate))
				for i, user := range usersToCreate {
					userProfileBatchInserters[i] = user
				}
				err = BatchInsert(db, "USER_PROFILE", userProfileBatchInserters, 500, log)
				if err != nil {
					log.Errorf("Error inserting user profiles: %v", err)
				}

				// Convert []Verse to []BatchInserter
				verseBatchInserters := make([]BatchInserter, len(versesToCreate))
				for i, verse := range versesToCreate {
					verseBatchInserters[i] = verse
				}

				err = BatchInsert(db, "VERSE", verseBatchInserters, 500, log)
				if err != nil {
					log.Errorf("Error inserting verses: %v", err)
				}

				// Convert []FileInfo to []BatchInserter
				fileInfoBatchInserters := make([]BatchInserter, len(fileInfosToCreate))
				for i, fileInfo := range fileInfosToCreate {
					fileInfoBatchInserters[i] = fileInfo
				}

				err = BatchInsert(db, "FILE_INFO", fileInfoBatchInserters, 500, log)
				if err != nil {
					log.Errorf("Error inserting file infos: %v", err)
				}

				// Convert []UserFollowing to []BatchInserter
				userFollowingBatchInserters := make([]BatchInserter, len(userFollowingToCreate))
				for i, userFollowing := range userFollowingToCreate {
					userFollowingBatchInserters[i] = userFollowing
				}

				err = BatchInsert(db, "USER_FOLLOWING", userFollowingBatchInserters, 500, log)
				if err != nil {
					log.Errorf("Error inserting user followings: %v", err)
				}

				time.Sleep(1 * time.Minute)
				offset = 0
				limit = 100
				// Clear the slices
				usersToCreate = nil
				versesToCreate = nil
				fileInfosToCreate = nil
				userFollowingToCreate = nil
				commitIds = make(map[string]bool)

				continue
			}

			for _, meta := range metaList {
				var count int
				var qry string

				found := false
				tableName, found := storverse.GetTableNameForAlias(meta.Alias, storverse.TypeConfigs)
				if found {
					log.Infof("tableName: %s", tableName)
					// Delete the row if the DATAID exists but the COMMITID does not match,
					// and return the COMMITID of the deleted row
					qry = fmt.Sprintf("DELETE FROM %s WHERE DATAID=? AND COMMITID<>? RETURNING COMMITID", tableName)
					row := db.QueryRowContext(ctx, qry, meta.DataId, meta.Commit)
					var commitIdToDelete string
					if err := row.Scan(&commitIdToDelete); err != nil {
						if err != sql.ErrNoRows {
							return nil, err
						}
					} else {
						// Log the number of rows deleted and proceed with the insert operation
						log.Infof("Deleted 1 row with COMMITID=%s from table %s", commitIdToDelete, tableName)
					}
					qry = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE COMMITID=? AND DATAID=?", tableName)
				} else {
					continue
				}

				row := db.QueryRowContext(ctx, qry, meta.Commit, meta.DataId)
				err = row.Scan(&count)
				if err != nil {
					return nil, err
				}

				if count > 0 {
					continue
				} else {
					log.Info("no existing record found, get data from gateway")
					if strings.Contains(platFormIds, meta.GroupId) && storverse.AliasInTypeConfigs(meta.Alias, storverse.TypeConfigs) {
						resp, err := getDataModel(ctx, didManager, meta.DataId, platFormIds, chainSvc, gatewayAddress, gatewayApi, log)
						if err != nil {
							continue
						}

						record, err := processMeta(meta, &resp, log)
						if err != nil {
							log.Errorf("failed to process meta: %w", err)
							continue
						}

						log.Infof("record: %v", record)
						switch r := record.(type) {
						case storverse.UserProfile:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to usersToCreate: %v", r)
								usersToCreate = append(usersToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.Verse:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to versesToCreate: %v", r)
								versesToCreate = append(versesToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.FileInfo:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to fileInfosToCreate: %v", r)
								fileInfosToCreate = append(fileInfosToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.UserFollowing:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to userFollowingToCreate: %v", r)
								userFollowingToCreate = append(userFollowingToCreate, r)
								commitIds[r.CommitID] = true
							}
						default:
							log.Warnf("unsupported record type: %T", r)
						}
					}
				}
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

func InitializeStorverseTables(ctx context.Context, log *logging.ZapEventLogger, db *sql.DB) {
	// initialize the user_profile database tables
	log.Info("creating user_profile tables...")
	if _, err := db.ExecContext(ctx, createUserProfileDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating user_profile tables done.")

	// initialize the verse database tables
	log.Info("creating verse tables...")
	if _, err := db.ExecContext(ctx, createVerseDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating verse tables done.")

	// initialize the file_info database tables
	log.Info("creating file_info tables...")
	if _, err := db.ExecContext(ctx, createFileInfoDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating file_info tables done.")

	// initialize the user_following database tables
	log.Info("creating user_following tables...")
	if _, err := db.ExecContext(ctx, createUserFollowingDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating user_following tables done.")
}

// // Define your function that accepts a context.Context as a parameter
func GetDidManager(ctx context.Context, keyName string, keyringHome string) (*did.DidManager, string, error) {
	address, err := chain.GetAddress(ctx, keyringHome, keyName)
	if err != nil {
		return nil, "", err
	}

	payload := fmt.Sprintf("cosmos %s allows to generate did", address)
	secret, err := chain.SignByAccount(ctx, keyringHome, keyName, []byte(payload))
	if err != nil {
		return nil, "", types.Wrap(types.ErrSignedFailed, err)
	}

	provider, err := saokey.NewSecp256k1Provider(secret)
	if err != nil {
		return nil, "", types.Wrap(types.ErrCreateProviderFailed, err)
	}
	resolver := saokey.NewKeyResolver()

	didManager := did.NewDidManager(provider, resolver)
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		return nil, "", types.Wrap(types.ErrAuthenticateFailed, err)
	}

	return &didManager, address, nil
}

func getDataModel(ctx context.Context, didManager *did.DidManager, dataId string, platFormIds string,
	chainSvc *chain.ChainSvc, gatewayAddress string, gatewayApi api.SaoApi, log *logging.ZapEventLogger) (apitypes.LoadResp, error) {
	proposal := saotypes.QueryProposal{
		Owner:   didManager.Id,
		Keyword: dataId,
		GroupId: platFormIds,
	}

	request, err := buildQueryRequest(ctx, didManager, proposal, chainSvc, gatewayAddress)
	if err != nil {
		log.Errorf("failed to build query request: %w", err)
		return apitypes.LoadResp{}, err
	}

	resp, err := gatewayApi.ModelLoad(ctx, request)
	if err != nil {
		log.Errorf("failed to load model: %w", err)
		return apitypes.LoadResp{}, err
	}
	log.Info(resp.Content)
	return resp, nil
}

func processMeta(meta modeltypes.Metadata, resp *apitypes.LoadResp, log *logging.ZapEventLogger) (BatchInserter, error) {
	for alias, config := range storverse.TypeConfigs {
		if strings.Contains(meta.Alias, alias) {
			recordPtr := reflect.New(config.RecordType)

			var raw map[string]interface{}
			if err := json.Unmarshal([]byte(resp.Content), &raw); err != nil {
				return nil, err
			}

			if fd, ok := raw["followingDataId"]; ok {
				switch v := fd.(type) {
				case string:
					if v != "" {
						raw["followingDataId"] = []string{v}
					} else {
						delete(raw, "followingDataId")
					}
				case []interface{}:
					var followingDataID []string
					for _, item := range v {
						if s, ok := item.(string); ok {
							followingDataID = append(followingDataID, s)
						}
					}
					raw["followingDataId"] = followingDataID
				}
			}

			updatedData, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(updatedData, recordPtr.Interface())
			if err != nil {
				log.Errorf("Unmarshal error: %v", err)
				return nil, err
			}

			log.Debug(recordPtr)

			record := recordPtr.Elem()

			// Set CommitID and DataID fields
			commitIDField := record.FieldByName("CommitID")
			if commitIDField.IsValid() && commitIDField.CanSet() {
				commitIDField.Set(reflect.ValueOf(meta.Commit))
			}
			dataIDField := record.FieldByName("DataID")
			if dataIDField.IsValid() && dataIDField.CanSet() {
				dataIDField.Set(reflect.ValueOf(meta.DataId))
			}
			aliasField := record.FieldByName("Alias")
			if aliasField.IsValid() && aliasField.CanSet() {
				aliasField.Set(reflect.ValueOf(meta.Alias))
			}

			log.Debugf("Processed record: %v", record)

			return record.Interface().(BatchInserter), nil
		}
	}

	return nil, fmt.Errorf("unsupported meta alias")
}

func buildQueryRequest(ctx context.Context, didManager *did.DidManager, proposal saotypes.QueryProposal, chain chain.ChainSvcApi, gatewayAddress string) (*types.MetadataProposal, error) {
	lastHeight, err := chain.GetLastHeight(ctx)
	if err != nil {
		return nil, types.Wrap(types.ErrQueryHeightFailed, err)
	}

	peerInfo, err := chain.GetNodePeer(ctx, gatewayAddress)
	if err != nil {
		return nil, err
	}

	proposal.LastValidHeight = uint64(lastHeight + 200)
	proposal.Gateway = peerInfo

	if proposal.Owner == "all" {
		return &types.MetadataProposal{
			Proposal: proposal,
		}, nil
	}

	proposalBytes, err := proposal.Marshal()
	if err != nil {
		return nil, types.Wrap(types.ErrMarshalFailed, err)
	}

	jws, err := didManager.CreateJWS(proposalBytes)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateJwsFailed, err)
	}

	return &types.MetadataProposal{
		Proposal: proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: jws.Signatures[0].Protected,
			Signature: jws.Signatures[0].Signature,
		},
	}, nil
}

func BatchInsert(db *sql.DB, tableName string, records []BatchInserter, batchSize int, log *logging.ZapEventLogger) error {
	if len(records) > 0 {
		valueArgs := ""
		log.Infof("Batch prepare, %d records to be created.", len(records))

		for index, record := range records {
			if valueArgs != "" {
				valueArgs += ", "
			}
			valueArgs += record.InsertValues()

			if index > 0 && index%batchSize == 0 {
				log.Infof("Sub batch prepare, %d records to be saved.", batchSize)
				stmt := fmt.Sprintf("INSERT INTO %s VALUES %s", tableName, valueArgs)
				_, err := db.Exec(stmt)
				if err != nil {
					return err
				}
				valueArgs = ""
				log.Infof("Sub batch done, %d records saved.", batchSize)
			}
		}

		if len(valueArgs) > 0 {
			stmt := fmt.Sprintf("INSERT INTO %s VALUES %s", tableName, valueArgs)
			log.Info(stmt)
			_, err := db.Exec(stmt)
			if err != nil {
				return err
			}
			log.Infof("Batch done, %d records saved.", len(records))
		}
	}
	return nil
}
