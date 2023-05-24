package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	did "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	logging "github.com/ipfs/go-log/v2"
	"io/ioutil"
	"os"
	"path/filepath"
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

//go:embed sqls/create_listing_info_table.sql
var createListingInfoDBSQL string

//go:embed sqls/create_purchase_order_table.sql
var createPurchaseOrderDBSQL string

//go:embed sqls/create_file_content_table.sql
var createFileContentDBSQL string

//go:embed sqls/create_verse_comment_table.sql
var createVerseCommentDBSQL string

//go:embed sqls/create_verse_comment_like_table.sql
var createVerseCommentLikeDBSQL string

//go:embed sqls/create_verse_like_table.sql
var createVerseLikeDBSQL string

//go:embed sqls/create_notification_table.sql
var createNotificationDBSQL string

//go:embed sqls/create_read_notifications_table.sql
var createReadNotificationsDBSQL string

type InsertionMap map[string]storverse.InsertionStrategy

var insertionStrategies = InsertionMap{
	"USER_PROFILE":    storverse.UserProfileInsertionStrategy{},
	"VERSE":           storverse.VerseInsertionStrategy{},
	"FILE_INFO":       storverse.FileInfoInsertionStrategy{},
	"USER_FOLLOWING":  storverse.UserFollowingInsertionStrategy{},
	"LISTING_INFO":    storverse.ListingInfoInsertionStrategy{},
	"PURCHASE_ORDER":  storverse.PurchaseOrderInsertionStrategy{},
	"VERSE_COMMENT":   storverse.VerseCommentInsertionStrategy{},
	"VERSE_COMMENT_LIKE": storverse.VerseCommentLikeInsertionStrategy{},
	"VERSE_LIKE":      storverse.VerseLikeInsertionStrategy{},
	"NOTIFICATION":    storverse.NotificationInsertionStrategy{},
	"READ_NOTIFICATIONS": storverse.ReadNotificationsInsertionStrategy{},
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
		token := os.Getenv("STORVERSE_TOKEN")
		if token == "" {
			log.Error("STORVERSE_TOKEN environment variable not set")
		}
		keyName := os.Getenv("STORVERSE_KEY_NAME")
		keyringHome := os.Getenv("SAO_KEYRING_HOME")
		log.Infof("keyName: %s, keyringHome: %s, token: %s", keyName, keyringHome, token)
		if keyName == "" || keyringHome == "" {
			log.Error("keyName or keyringHome is empty, please set the env variable")
			return nil, errors.New("keyName or keyringHome is empty")
		}
		gatewayApi, closer, err := apiclient.NewNodeApi(ctx, gwAddress, token)
		if err != nil {
			log.Errorf("failed to get gateway api: %w", err)
			return nil, err
		}
		defer closer()

		stagingHome := os.Getenv("STORVERSE_STAGING_HOME")
		if stagingHome == "" {
			log.Warn("STAGING_HOME environment variable not set")
			return nil, errors.New("STAGING_HOME environment variable not set")
		}
		didManager, _, err := GetDidManager(ctx, keyName, keyringHome)
		if err != nil {
			log.Errorf("failed to get did manager: %w", err)
			return nil, errors.New("failed to get did manager")
		}

		gatewayAddress, err := gatewayApi.GetNodeAddress(ctx)
		if err != nil {
			log.Errorf("failed to get gateway address: %w", err)
			return nil, errors.New("failed to get gateway address")
		}

		//s := gocron.NewScheduler(time.UTC)

		// Schedule the function to run every 1 hour
		//_, err = s.Every(1).Seconds().Do(sync.UpdateEthAddresses(db, log))
		//if err != nil {
		//	log.Errorf("failed to schedule job: %w", err)
		//	return nil, errors.New("failed to schedule job")
		//}

		var offset uint64 = 0
		var limit uint64 = 100
		// Create slice to store the data
		var usersToCreate []storverse.UserProfile
		var versesToCreate []storverse.Verse
		var fileInfosToCreate []storverse.FileInfo
		var userFollowingToCreate []storverse.UserFollowing
		var listingInfosToCreate []storverse.ListingInfo
		var purchaseOrdersToCreate []storverse.PurchaseOrder
		var verseCommentsToCreate []storverse.VerseComment
		var verseCommentLikesToCreate []storverse.VerseCommentLike
		var verseLikesToCreate []storverse.VerseLike
		var notificationsToCreate []storverse.Notification
		var readNotificationsToCreate []storverse.ReadNotifications
		commitIds := make(map[string]bool)

		errorMap := make(map[string]int)
		filterMap := make(map[string]time.Time)
		filterCountMap := make(map[string]int)

		for {
			metaList, total, err := chainSvc.ListMeta(ctx, offset*limit, limit)
			log.Debugf("offset: %d, limit: %d, total: %d", offset*limit, limit, total)
			if err != nil {
				return nil, err
			}
			if offset*limit <= total {
				offset++
			} else {
				// Create the items map
				var itemsMap = map[string][]interface{}{
					"USER_PROFILE":       convertToInterfaceSlice(usersToCreate),
					"VERSE":              convertToInterfaceSlice(versesToCreate),
					"FILE_INFO":          convertToInterfaceSlice(fileInfosToCreate),
					"USER_FOLLOWING":     convertToInterfaceSlice(userFollowingToCreate),
					"LISTING_INFO":       convertToInterfaceSlice(listingInfosToCreate),
					"PURCHASE_ORDER":     convertToInterfaceSlice(purchaseOrdersToCreate),
					"VERSE_COMMENT":      convertToInterfaceSlice(verseCommentsToCreate),
					"VERSE_COMMENT_LIKE": convertToInterfaceSlice(verseCommentLikesToCreate),
					"VERSE_LIKE":         convertToInterfaceSlice(verseLikesToCreate),
					"NOTIFICATION":       convertToInterfaceSlice(notificationsToCreate),
					"READ_NOTIFICATIONS": convertToInterfaceSlice(readNotificationsToCreate),
				}

				// Iterate over the strategies and call performBatchInsert for each type
				for typeName, strategy := range insertionStrategies {
					items := itemsMap[typeName]
					if err := performBatchInsert(db, strategy, items, 500, log); err != nil {
						log.Errorf("Error inserting %s: %v", typeName, err)
					}
				}

				rowsAffected, err := storverse.UpdateUserFollowingStatus(ctx, db)
				if err != nil {
					log.Errorf("Error updating USER_FOLLOWING records: %v", err)
				} else {
					log.Infof("Updated %d rows in USER_FOLLOWING", rowsAffected)
				}

				// update notification read status
				rowsAffected, err = storverse.UpdateNotificationReadStatus(ctx, db)
				if err != nil {
					log.Errorf("Error updating NOTIFICATION records: %v", err)
				} else {
					log.Infof("Updated %d rows in NOTIFICATION", rowsAffected)
				}

				time.Sleep(2 * time.Second)
				offset = 0
				limit = 100
				// Clear the slices
				usersToCreate = nil
				versesToCreate = nil
				fileInfosToCreate = nil
				userFollowingToCreate = nil
				listingInfosToCreate = nil
				purchaseOrdersToCreate = nil
				verseCommentsToCreate = nil
				verseCommentLikesToCreate = nil
				verseLikesToCreate = nil
				notificationsToCreate = nil
				readNotificationsToCreate = nil
				commitIds = make(map[string]bool)

				continue
			}

			for _, meta := range metaList {
				// Check if meta.DataId is in filterMap and if the timeout has not passed
				if filterTime, ok := filterMap[meta.DataId]; ok && time.Since(filterTime) < getTimeoutDuration(filterCountMap[meta.DataId]) {
					continue
				}

				var count int
				var qry string

				found := false
				tableName, found := storverse.GetTableNameForAlias(meta.Alias, storverse.TypeConfigs)
				if found {
					log.Debugf("tableName: %s", tableName)
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
					if strings.Contains(platFormIds, meta.GroupId) && (storverse.AliasInTypeConfigs(meta.Alias, storverse.TypeConfigs) || strings.Contains(meta.Alias, "filecontent")) {
						resp, err := getDataModel(ctx, didManager, meta.DataId, meta.Commit, platFormIds, chainSvc, gatewayAddress, gatewayApi, log)
						if err != nil {
							// Increment error count for meta.DataId
							errorMap[meta.DataId]++

							// If error count reaches 10, add meta.DataId to filterMap, increment filter count, and reset error count
							if errorMap[meta.DataId] >= 10 {
								filterMap[meta.DataId] = time.Now()
								filterCountMap[meta.DataId]++
								errorMap[meta.DataId] = 0
							}

							continue
						}

						// Reset error count if getDataModel is successful
						delete(errorMap, meta.DataId)
						delete(filterMap, meta.DataId)

						//if meta.Alias contains filecontent, save the resp content to a file under keyringHome/tmp folder
						if strings.Contains(meta.Alias, "filecontent") {
							log.Info("save file content to a file")
							fileName := meta.DataId
							filePath := filepath.Join(stagingHome, fileName)
							if _, err := os.Stat(filePath); os.IsNotExist(err) {
								os.MkdirAll(stagingHome, os.ModePerm)
							}
							log.Info("file path: ", filePath)

							// Remove leading and trailing quotation marks from resp.Content
							content := strings.Trim(resp.Content, "\"")

							// write file
							err := ioutil.WriteFile(filePath, []byte(content), 0644)
							if err != nil {
								log.Errorf("failed to write file: %w", err)
								continue
							}

							// insert a record to FILE_CONTENT table
							qry := fmt.Sprintf("INSERT INTO FILE_CONTENT (COMMITID, DATAID, CONTENTPATH, ALIAS, CREATEDAT, OWNER) VALUES (?, ?, ?, ?, ?, ?)")
							_, err = db.ExecContext(ctx, qry, meta.Commit, meta.DataId, filePath, meta.Alias, meta.CreatedAt, meta.Owner)
							if err != nil {
								log.Errorf("failed to insert file content: %w", err)
								continue
							}
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
								// Create a notification for the user being followed
								notification, err, skip := storverse.CreateNotification(db, record)
								if err != nil {
									log.Errorf("Error creating notification: %v", err)
								} else {
									notificationsToCreate = append(notificationsToCreate, *notification)
								}
								if skip {
									continue
								}

								log.Infof("add to userFollowingToCreate: %v", r)
								userFollowingToCreate = append(userFollowingToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.ListingInfo:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to listingInfosToCreate: %v", r)
								listingInfosToCreate = append(listingInfosToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.PurchaseOrder:
							if _, ok := commitIds[r.CommitID]; !ok {
								// Create a notification for the purchase order
								notification, err, skip := storverse.CreateNotification(db, record)
								if err != nil {
									log.Errorf("Error creating notification: %v", err)
								} else {
									notificationsToCreate = append(notificationsToCreate, *notification)
								}
								if skip {
									continue
								}

								log.Infof("add to purchaseOrdersToCreate: %v", r)
								purchaseOrdersToCreate = append(purchaseOrdersToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.VerseComment:
							if _, ok := commitIds[r.CommitID]; !ok {
								// Create a notification for the verse comment
								notification, err, skip := storverse.CreateNotification(db, record)
								if err != nil {
									log.Errorf("Error creating notification: %v", err)
								} else {
									notificationsToCreate = append(notificationsToCreate, *notification)
								}
								if skip {
									continue
								}

								log.Infof("add to verseCommentsToCreate: %v", r)
								verseCommentsToCreate = append(verseCommentsToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.VerseCommentLike:
							if _, ok := commitIds[r.CommitID]; !ok {
								// Create a notification for the verse comment like
								notification, err, skip := storverse.CreateNotification(db, record)
								if err != nil {
									log.Errorf("Error creating notification: %v", err)
								} else {
									notificationsToCreate = append(notificationsToCreate, *notification)
								}
								if skip {
									continue
								}

								log.Infof("add to verseCommentLikesToCreate: %v", r)
								verseCommentLikesToCreate = append(verseCommentLikesToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.VerseLike:
							if _, ok := commitIds[r.CommitID]; !ok {
								// Create a notification for the verse like
								notification, err, skip := storverse.CreateNotification(db, record)
								if err != nil {
									log.Errorf("Error creating notification: %v", err)
								} else {
									notificationsToCreate = append(notificationsToCreate, *notification)
								}
								if skip {
									continue
								}

								log.Infof("add to verseLikesToCreate: %v", r)
								verseLikesToCreate = append(verseLikesToCreate, r)
								commitIds[r.CommitID] = true
							}
						case storverse.ReadNotifications:
							if _, ok := commitIds[r.CommitID]; !ok {
								log.Infof("add to readNotificationsToCreate: %v", r)
								readNotificationsToCreate = append(readNotificationsToCreate, r)
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

	// initialize the listing_info database tables
	log.Info("creating listing_info tables...")
	if _, err := db.ExecContext(ctx, createListingInfoDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating listing_info tables done.")

	// initialize the purchase_order database tables
	log.Info("creating purchase_order tables...")
	if _, err := db.ExecContext(ctx, createPurchaseOrderDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating purchase_order tables done.")

	// initialize the file_content database tables
	log.Info("creating file_content tables...")
	if _, err := db.ExecContext(ctx, createFileContentDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating file_content tables done.")

	// initialize the verse_comment database tables
	log.Info("creating verse_comment tables...")
	if _, err := db.ExecContext(ctx, createVerseCommentDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating verse_comment tables done.")

	// initialize the verse_comment_like database tables
	log.Info("creating verse_comment_like tables...")
	if _, err := db.ExecContext(ctx, createVerseCommentLikeDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating verse_comment_like tables done.")

	// initialize the verse_like database tables
	log.Info("creating verse_like tables...")
	if _, err := db.ExecContext(ctx, createVerseLikeDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating verse_like tables done.")

	// initialize the notification database tables
	log.Info("creating notification tables...")
	if _, err := db.ExecContext(ctx, createNotificationDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating notification tables done.")

	// initialize the read_notifications database tables
	log.Info("creating read_notifications tables...")
	if _, err := db.ExecContext(ctx, createReadNotificationsDBSQL); err != nil {
		log.Errorf("failed to create tables: %w", err)
	}
	log.Info("creating read_notifications tables done.")
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

func getDataModel(ctx context.Context, didManager *did.DidManager, dataId string, commitId string, platFormIds string,
	chainSvc *chain.ChainSvc, gatewayAddress string, gatewayApi api.SaoApi, log *logging.ZapEventLogger) (apitypes.LoadResp, error) {
	proposal := saotypes.QueryProposal{
		Owner:   didManager.Id,
		Keyword: dataId,
		GroupId: platFormIds,
		CommitId:  commitId,
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

func processMeta(meta modeltypes.Metadata, resp *apitypes.LoadResp, log *logging.ZapEventLogger) (storverse.BatchInserter, error) {
	if config, found := storverse.GetMatchingTypeConfig(meta.Alias, storverse.TypeConfigs); found {
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
			if !strings.Contains(err.Error(), "cannot unmarshal") || !strings.Contains(err.Error(), "into Go struct field") {
				return nil, err
			}
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

		return record.Interface().(storverse.BatchInserter), nil
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

func convertToInterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		log.Error("convertToInterfaceSlice() given a non-slice type")
		return nil
	}

	result := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		result[i] = s.Index(i).Interface()
	}
	return result
}

func performBatchInsert(db *sql.DB, strategy storverse.InsertionStrategy, items []interface{}, batchSize int, log *logging.ZapEventLogger) error {
	batchInserters := make([]storverse.BatchInserter, len(items))
	for i, item := range items {
		batchInserters[i] = strategy.Convert(item)
	}

	return BatchInsert(db, strategy.TableName(), batchInserters, batchSize, log)
}

func BatchInsert(db *sql.DB, tableName string, records []storverse.BatchInserter, batchSize int, log *logging.ZapEventLogger) error {
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

// Returns the timeout duration based on the filter count
func getTimeoutDuration(filterCount int) time.Duration {
	if filterCount == 1 {
		return 2 * time.Minute
	} else if filterCount >= 2 && filterCount <= 10 {
		return 5 * time.Minute
	} else {
		return time.Hour
	}
}
