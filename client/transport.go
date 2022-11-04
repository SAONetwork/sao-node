package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sao-storage-node/node/utils"
	"sao-storage-node/types"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/mitchellh/go-homedir"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport-client")

const SAO_CLI_KEY_PATH = "~/.sao_cli_key/"

func DoTransport(ctx context.Context, remoteAddr string, remotePeerId string, fpath string) cid.Cid {
	file, err := os.Open(fpath)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	serverAddress, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	serverId, err := peer.Decode(remotePeerId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	clientKey := fetchKey()
	if clientKey == nil {
		log.Error("failed to generate transport key")
		return cid.Undef
	}

	tr, err := libp2pwebtransport.New(clientKey, nil, network.NullResourceManager)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	log.Info("Dialing ", serverId, " (", serverAddress, ")")
	conn, err := tr.Dial(ctx, serverAddress, serverId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	defer conn.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	contentCid, err := utils.CaculateCid(content)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	var contentLength int = len(content)
	var totalChunks = contentLength/types.CHUNK_SIZE + 1
	chunkId := 0
	for chunkId <= totalChunks {
		var chunk []byte
		if (chunkId+1)*types.CHUNK_SIZE < len(content) {
			chunk = content[chunkId*types.CHUNK_SIZE : (chunkId+1)*types.CHUNK_SIZE]
		} else if chunkId*types.CHUNK_SIZE < len(content) {
			chunk = content[chunkId*types.CHUNK_SIZE:]
		} else {
			chunk = make([]byte, 0)
		}

		chunkCid, err := utils.CaculateCid(chunk)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		log.Info("Content[", chunkId, "], CID: ", chunkCid, ", length: ", len(chunk))

		str, err := conn.OpenStream(ctx)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}
		defer str.Close()

		req := &types.FileChunkReq{
			ChunkId:     chunkId,
			TotalLength: contentLength,
			TotalChunks: totalChunks,
			ChunkCid:    chunkCid.String(),
			Cid:         contentCid.String(),
			Content:     chunk,
		}
		bytes, err := json.Marshal(req)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		if _, err := str.Write(bytes); err != nil {
			log.Error(err)
			return cid.Undef
		}
		if err := str.CloseWrite(); err != nil {
			log.Error(err)
			return cid.Undef
		}

		res, err := io.ReadAll(str)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}
		remoteCid := string(res)

		if remoteCid == chunkCid.String() {
			chunkId++
		} else if remoteCid == contentCid.String() && len(chunk) == 0 {
			break
		} else {
			log.Error("file cid mismatch, ", chunkCid, " vs. ", remoteCid)
			return cid.Undef
		}
	}

	return contentCid
}

func fetchKey() ic.PrivKey {
	kstorePath, err := homedir.Expand(SAO_CLI_KEY_PATH)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	keyPath := filepath.Join(kstorePath, "libp2p.key")
	key, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(kstorePath, 0700) //nolint: gosec
			if err != nil && !os.IsExist(err) {
				log.Error(err.Error())
				return nil
			}

			pk, _, err := ic.GenerateEd25519Key(rand.Reader)
			if err != nil {
				log.Error(err.Error())
				return nil
			}

			keyBytes, err := ic.MarshalPrivateKey(pk)
			if err != nil {
				log.Error(err.Error())
				return nil
			}

			err = os.WriteFile(keyPath, keyBytes, 0600)
			if err != nil {
				log.Error(err.Error())
				return nil
			}

			return pk
		} else {
			log.Error(err.Error())
			return nil
		}
	}

	pk, err := ic.UnmarshalPrivateKey(key)
	if err != nil {
		log.Error(err.Error())
		return nil
	} else {
		return pk
	}
}
