package gql

import (
	"context"
	"github.com/SaoNetwork/sao-did/sid"
	"net/http"
	"strings"
	"time"

	saodid "github.com/SaoNetwork/sao-did"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	"github.com/SaoNetwork/sao-node/chain"
)

func authenticate(next http.Handler, resolver *resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		// If token is empty, just proceed without authentication
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Try to extract signatures part from the token
		tokenParts := strings.Split(token, ":")
		if len(tokenParts) < 5 {
			// If token format is incorrect, just proceed without authentication
			next.ServeHTTP(w, r)
			return
		}
		signatures := tokenParts[4]

		// Try to get the claims from the cache
		cachedClaims, found := resolver.cache.Get(signatures)
		if found && cachedClaims != nil {
			// If the signatures is cached, proceed with the request and pass the cached claims
			ctx := context.WithValue(r.Context(), "claims", cachedClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If not found in cache, validate the token
		claims, valid := validateToken(token, r.Context(), resolver)
		if valid {
			// Store the valid claims in cache for 10 minutes
			resolver.cache.Set(signatures, claims, 10*time.Minute)

			// Pass the claims to the request context and proceed
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}


func validateToken(tokenString string, ctx context.Context, resolver *resolver) (string, bool) {
	if len(strings.Split(tokenString, " ")) == 2 {
		tokenString = strings.Split(tokenString, " ")[1]
	}
	// get owner from tokenString, split by ":" and the first three parts is owner
	owner := strings.Split(tokenString, ":")[0] + ":" + strings.Split(tokenString, ":")[1] + ":" + strings.Split(tokenString, ":")[2]

	// get protected from tokenString, split by ":" and the fourth part is protected
	protected := strings.Split(tokenString, ":")[3]

	// get signatures from tokenString, split by ":" and the fifth part is signatures
	signatures := strings.Split(tokenString, ":")[4]

	didManager, err := saodid.NewDidManagerWithDid(owner, getSidDocFunc(ctx, resolver.chainSvc))
	if err != nil {
		return "", false
	}

	signature := saodidtypes.JwsSignature{
		Protected: protected,
		Signature: signatures,
	}

	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: owner,
		Signatures: []saodidtypes.JwsSignature{
			signature,
		},
	})
	if err != nil {
		return "", false
	}

	var dataId string
	err = resolver.indexSvc.Db.QueryRowContext(ctx, "SELECT DATAID FROM USER_PROFILE WHERE DID = ?", owner).Scan(&dataId)
	if err != nil {
		return "", false
	}

	return dataId, true
}

func getSidDocFunc(ctx context.Context, chainSvc *chain.ChainSvc) func(versionId string) (*sid.SidDocument, error) {
	return func(versionId string) (*sid.SidDocument, error) {
		return chainSvc.GetSidDocument(ctx, versionId)
	}
}