package gql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"io/ioutil"
	"sao-node/node/indexer/gql/types"
	"strconv"
)

type fileInfo struct {
	CommitId        string
	DataID          string
	Alias           string
	CreatedAt       types.Uint64
	FileDataID      string
	ContentType     string
	Owner           string
	Filename        string
	FileCategory    string
	ExtendInfo      string
	ThumbnailDataId string
	VerseId         string
}

type fileContent struct {
	CommitId    string
	DataID      string
	Alias       string
	CreatedAt   types.Uint64
	Owner       string
	ContentPath string
}

type FileInfoResult struct {
	FileInfos []*fileInfo
	HasMore   bool
}

// query: fileInfo(id, userDataId) FileInfo
func (r *resolver) FileInfo(ctx context.Context, args struct {
	ID         graphql.ID
	UserDataId *string
}) (*fileInfo, error) {
	claims, ok := ctx.Value("claims").(string)
	// If UserDataId is not nil, require it to match the claims
	if args.UserDataId != nil && (!ok || claims != *args.UserDataId) {
		return nil, errors.New("Unauthorized")
	}

	var dataId uuid.UUID
	err := dataId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	var fi fileInfo
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_INFO WHERE DATAID = ?", dataId)
	err = row.Scan(
		&fi.CommitId,
		&fi.DataID,
		&fi.Alias,
		&fi.CreatedAt,
		&fi.FileDataID,
		&fi.ContentType,
		&fi.Owner,
		&fi.Filename,
		&fi.FileCategory,
		&fi.ExtendInfo,
		&fi.ThumbnailDataId,
	)
	if err != nil {
		return nil, err
	}

	// Find verse by fileInfo ID
	var v verse
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE FILEIDS LIKE ?", "%"+fi.CommitId+"%").Scan(
		&v.CommitId,
		&v.DataId,
		&v.Alias,
		&v.CreatedAt,
		&v.FileIDs,
		&v.Owner,
		&v.Price,
		&v.Digest,
		&v.Scope,
		&v.Status,
		&v.NftTokenID,
		&v.FileType,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return &fi, nil
	}

	// Set the VerseId
	fi.VerseId = v.DataId

	// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
	if v.Price != "" {
		price, err := strconv.ParseFloat(v.Price, 64)
		if err != nil {
			return nil, err
		}

		if price > 0 {
			var count int
			if args.UserDataId != nil {
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			}
			if err != nil {
				return nil, err
			}

			v.IsPaid = count > 0
		}
	}

	if args.UserDataId != nil {
		// Process verse scope
		_, err = processVerseScope(ctx, r.indexSvc.Db, &v, *args.UserDataId)
		if err != nil {
			return nil, err
		}

		// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
		if v.Price != "" {
			price, err := strconv.ParseFloat(v.Price, 64)
			if err != nil {
				return nil, err
			}

			if price > 0 {
				var count int
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
				if err != nil {
					return nil, err
				}

				if count == 0 {
					return nil, errors.New("the file is charged and not paid yet")
				}
			}
		}
	} else {
		if v.Scope == 2 || v.Scope == 3 || v.Scope == 4 {
			return nil, errors.New("you are not authorized to access the file")
		}
		if v.Scope == 5 {
			return nil, errors.New("the file is private")
		}
	}

	return &fi, nil
}

func (r *resolver) FileInfosByVerseIds(ctx context.Context, args struct {
	VerseIds []string
	UserDataId *string
}) ([]*fileInfo, error) {
	var fileInfos []*fileInfo

	// Fetch the fileIDs, Scope, Owner, CreatedAt for each verseId in the order of verseIds
	orderedFileIDs := make([]string, 0)
	orderedVerseIDs := make([]string, 0) // To keep track of verseId for each fileId
	for _, verseId := range args.VerseIds {
		rows, err := r.indexSvc.Db.QueryContext(ctx, "SELECT FILEIDS, SCOPE, OWNER, CREATEDAT FROM VERSE WHERE DATAID = ? AND FILETYPE = 'image'", verseId)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var fileIDsJSON string
		var scope int32
		var owner string
		var createdAt types.Uint64
		if rows.Next() {
			err = rows.Scan(&fileIDsJSON, &scope, &owner, &createdAt)
			if err != nil {
				return nil, err
			}

			var count int
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", verseId, *args.UserDataId).Scan(&count)
			if err != nil {
				return nil, err
			}

			v := &verse{
				DataId:    verseId,
				Scope:     scope,
				Owner:     owner,
				CreatedAt: createdAt,
				IsPaid:    count > 0,
			}

			// Process verse scope
			v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
			if err != nil {
				// print error and continue
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
			if v.NotInScope > 1 {
				// verse is not accessible
				fmt.Printf("verse is not accessible: %s\n", verseId)
				continue
			}

			var fileIDs []string
			err = json.Unmarshal([]byte(fileIDsJSON), &fileIDs)
			if err != nil {
				return nil, fmt.Errorf("parsing FILEIDS '%s' as JSON: %w", fileIDsJSON, err)
			}

			orderedFileIDs = append(orderedFileIDs, fileIDs...)
			for range fileIDs {
				orderedVerseIDs = append(orderedVerseIDs, verseId)
			}
		}
	}

	// Fetch the fileInfo for each fileID and create a map of fileID to fileInfo
	fileInfoMap := make(map[string]*fileInfo)
	for i, fileID := range orderedFileIDs {
		row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_INFO WHERE DATAID = ?", fileID)

		var fi fileInfo
		err := row.Scan(
			&fi.CommitId,
			&fi.DataID,
			&fi.Alias,
			&fi.CreatedAt,
			&fi.FileDataID,
			&fi.ContentType,
			&fi.Owner,
			&fi.Filename,
			&fi.FileCategory,
			&fi.ExtendInfo,
			&fi.ThumbnailDataId,
		)
		if err != nil {
			return nil, err
		}

		// Add verseId field to fileInfo
		fi.VerseId = orderedVerseIDs[i]
		fileInfoMap[fileID] = &fi
	}

	// Order the fileInfos according to the order of fileIDs in orderedFileIDs and limit the size to 10
	for _, fileID := range orderedFileIDs {
		if len(fileInfos) >= 12 {
			break
		}

		if fi, ok := fileInfoMap[fileID]; ok {
			fileInfos = append(fileInfos, fi)
		}
	}

	return fileInfos, nil
}

func (r *resolver) FileInfos(ctx context.Context, args struct {
	UserDataId string
	Limit      *int32
	Offset     *int32
	Caller     *string
}) (*FileInfoResult, error) {
	// Default limit is 10 and offset is 0
	limit := 10
	offset := 0
	if args.Limit != nil {
		limit = int(*args.Limit)
	}
	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	// Prepare the base query
	query := "SELECT * FROM FILE_INFO WHERE OWNER = ? ORDER BY CREATEDAT DESC LIMIT ? OFFSET ?"

	// Execute the query, fetch one more row than the limit
	rows, err := r.indexSvc.Db.QueryContext(ctx, query, args.UserDataId, limit+1, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fileInfos []*fileInfo
	var count int
	for rows.Next() {
		count++
		// If count exceeds limit, break from loop, indicating there is more data
		if count > limit {
			break
		}
		var fi fileInfo
		err := rows.Scan(
			&fi.CommitId,
			&fi.DataID,
			&fi.Alias,
			&fi.CreatedAt,
			&fi.FileDataID,
			&fi.ContentType,
			&fi.Owner,
			&fi.Filename,
			&fi.FileCategory,
			&fi.ExtendInfo,
			&fi.ThumbnailDataId,
		)
		if err != nil {
			return nil, err
		}

		v := &verse{}
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE FILEIDS LIKE ?", "%"+fi.CommitId+"%").Scan(
			&v.CommitId,
			&v.DataId,
			&v.Alias,
			&v.CreatedAt,
			&v.FileIDs,
			&v.Owner,
			&v.Price,
			&v.Digest,
			&v.Scope,
			&v.Status,
			&v.NftTokenID,
			&v.FileType,
		)

		if err == sql.ErrNoRows {
			// if args.Caller equals to args.UserDataId, then the verse is in scope
			if args.Caller != nil && *args.Caller == args.UserDataId {
				fileInfos = append(fileInfos, &fi)
			}
			continue
		} else if err != nil {
			fmt.Printf("error fetching verse: %s\n", err)
			continue
		} else if v.Status == "2" {
			fmt.Printf("verse has been deleted: %s\n", v.DataId)
			continue
		}

		if args.Caller != nil { // Process verse scope
			v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.Caller)
			if err != nil {
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
			if v.NotInScope > 1 {
				// verse is not accessible
				fmt.Printf("verse is not accessible: %s\n", v.DataId)
				continue
			}
		}

		// Set the VerseId
		fi.VerseId = v.DataId

		fileInfos = append(fileInfos, &fi)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Prepare result
	result := &FileInfoResult{
		FileInfos: fileInfos,
		HasMore:   count > limit,
	}

	return result, nil
}


// query: file(id, userDataId) String
func (r *resolver) File(ctx context.Context, args struct {
	ID              graphql.ID
	UserDataId      *string
	GetFromFileInfo *bool
}) (*string, error) {
	claims, ok := ctx.Value("claims").(string)
	// If UserDataId is not nil, require it to match the claims
	if args.UserDataId != nil && (!ok || claims != *args.UserDataId) {
		return nil, errors.New("Unauthorized")
	}

	var dataId uuid.UUID
	err := dataId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	var fc fileContent
	var fileDataID uuid.UUID
	var row *sql.Row

	if args.GetFromFileInfo != nil && *args.GetFromFileInfo {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT FILEDATAID FROM FILE_INFO WHERE DATAID = ?", dataId)
		err = row.Scan(&fileDataID)
		if err != nil {
			return nil, err
		}
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_CONTENT WHERE DATAID = ?", fileDataID)
	} else {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_CONTENT WHERE DATAID = ?", dataId)
	}

	err = row.Scan(
		&fc.CommitId,
		&fc.DataID,
		&fc.Alias,
		&fc.CreatedAt,
		&fc.Owner,
		&fc.ContentPath,
	)
	if err != nil {
		return nil, err
	}

	fileContentBytes, err := ioutil.ReadFile(fc.ContentPath)
	if err != nil {
		return nil, err
	}
	fileContentString := string(fileContentBytes)
	return &fileContentString, nil
}


func (fi *fileInfo) ID() graphql.ID {
	return graphql.ID(fi.DataID)
}
