package sync

import (
	"database/sql"
	"encoding/json"
	logging "github.com/ipfs/go-log/v2"
	"net/http"
	"strings"
)

type AccountListResponse struct {
	AccountList struct {
		Did         string   `json:"did"`
		AccountDids []string `json:"accountDids"`
	} `json:"accountList"`
}

type AccountIdResponse struct {
	AccountId struct {
		AccountDid string `json:"accountDid"`
		AccountId  string `json:"accountId"`
		Creator    string `json:"creator"`
	} `json:"accountId"`
}

func UpdateEthAddresses(db *sql.DB, log *logging.ZapEventLogger) error {
	type PendingUpdate struct {
		did      string
		ethAddrs string
	}

	var updates []PendingUpdate

	rows, err := db.Query("SELECT DID, ETHADDR FROM USER_PROFILE")
	if err != nil {
		log.Errorf("Error while querying DID and ETHADDR from USER_PROFILE: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var did, currentEthAddr string
		if err := rows.Scan(&did, &currentEthAddr); err != nil {
			log.Errorf("Error while scanning DID and ETHADDR: %v", err)
			continue
		}

		log.Debugf("Processing DID %s with current ETHADDR %s", did, currentEthAddr)

		accountListResp, err := getAccountList(did)
		if err != nil {
			log.Errorf("Error while fetching account list: %v", err)
			continue
		}

		var ethAddrs []string
		for _, accountDid := range accountListResp.AccountList.AccountDids {
			accountIdResp, err := getAccountId(accountDid)
			if err != nil {
				log.Errorf("Error while fetching account ID: %v", err)
				break
			}

			if strings.HasPrefix(accountIdResp.AccountId.AccountId, "eip155:97:") {
				ethAddr := strings.TrimPrefix(accountIdResp.AccountId.AccountId, "eip155:97:")
				ethAddrs = append(ethAddrs, ethAddr)
			}
		}

		newEthAddr := strings.Join(ethAddrs, ",")
		if newEthAddr != currentEthAddr {
			updates = append(updates, PendingUpdate{did, newEthAddr})
		}
	}

	if err := rows.Err(); err != nil {
		log.Errorf("Error while iterating over rows: %v", err)
		return err
	}

	for _, update := range updates {
		log.Infof("Updating ETHADDR for DID %s to %s", update.did, update.ethAddrs)
		_, err = db.Exec("UPDATE USER_PROFILE SET ETHADDR = ? WHERE DID = ?", update.ethAddrs, update.did)
		if err != nil {
			log.Errorf("Error while updating ETHADDR in USER_PROFILE: %v", err)
			continue
		}
	}

	return nil
}

func getAccountList(did string) (*AccountListResponse, error) {
	resp, err := http.Get("http://127.0.0.1:1317/SaoNetwork/sao/did/account_list/" + did + ":")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accountListResp AccountListResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountListResp); err != nil {
		return nil, err
	}

	return &accountListResp, nil
}

func getAccountId(accountDid string) (*AccountIdResponse, error) {
	resp, err := http.Get("http://127.0.0.1:1317/SaoNetwork/sao/did/account_id/" + accountDid + ":")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accountIdResp AccountIdResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountIdResp); err != nil {
		return nil, err
	}

	return &accountIdResp, nil
}
