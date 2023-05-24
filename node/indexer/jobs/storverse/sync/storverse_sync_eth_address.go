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
	// Fetch all DIDs and ETHADDR from the USER_PROFILE table
	rows, err := db.Query("SELECT DID, ETHADDR FROM USER_PROFILE")
	if err != nil {
		log.Errorf("Error while querying DID and ETHADDR from USER_PROFILE: %v", err)
		return err
	}
	defer rows.Close()

	// Loop through all rows
	for rows.Next() {
		var did, currentEthAddr string
		if err := rows.Scan(&did, &currentEthAddr); err != nil {
			log.Errorf("Error while scanning DID and ETHADDR: %v", err)
			continue
		}

		log.Infof("Processing DID %s with current ETHADDR %s", did, currentEthAddr)

		// Fetch the list of account DIDs associated with the current DID
		accountListResp, err := getAccountList(did)
		if err != nil {
			log.Errorf("Error while fetching account list: %v", err)
			continue
		}

		var ethAddrs []string
		hasError := false
		for _, accountDid := range accountListResp.AccountList.AccountDids {
			// Fetch the account ID for the current account DID
			accountIdResp, err := getAccountId(accountDid)
			if err != nil {
				log.Errorf("Error while fetching account ID: %v", err)
				hasError = true
				break
			}

			if strings.HasPrefix(accountIdResp.AccountId.AccountId, "eip155:97:") {
				ethAddr := strings.TrimPrefix(accountIdResp.AccountId.AccountId, "eip155:97:")
				ethAddrs = append(ethAddrs, ethAddr)
			}
		}

		if hasError {
			continue
		}

		// Join the ETH addresses with a comma
		newEthAddr := strings.Join(ethAddrs, ",")

		// If the ETHADDR fetched from the API is different from the ETHADDR in the USER_PROFILE table, update it
		if newEthAddr != currentEthAddr {
			log.Infof("Updating ETHADDR for DID %s from %s to %s", did, currentEthAddr, newEthAddr)
			_, err = db.Exec("UPDATE USER_PROFILE SET ETHADDR = ? WHERE DID = ?", newEthAddr, did)
			if err != nil {
				log.Errorf("Error while updating ETHADDR in USER_PROFILE: %v", err)
				continue
			}
		}
	}

	return rows.Err()
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
