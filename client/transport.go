package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/mitchellh/go-homedir"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport-client")

func DoTransport(ctx context.Context, repo string, remoteAddr string, remotePeerId string, fpath string) cid.Cid {
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

	clientKey := fetchKey(repo)
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

	contentCid, err := utils.CalculateCid(content)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	rpcReq := types.RpcReq{
		Method: "Sao.Upload",
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

		chunkCid, err := utils.CalculateCid(chunk)
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
		b, err := json.Marshal(req)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		rpcReq.Params = append(make([]string, 0), string(b))
		bytes, err := json.Marshal(rpcReq)
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

		buf, err := io.ReadAll(str)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		var resp types.RpcResp
		err = json.Unmarshal(buf, &resp)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		if resp.Error != "" {
			log.Error("resp err: ", resp.Error)
			return cid.Undef
		}

		remoteCid := resp.Data

		if remoteCid == chunkCid.String() {
			chunkId++
		} else if remoteCid == contentCid.String() && len(chunk) == 0 {
			break
		} else {
			log.Errorf("file cid mismatch, expected %s, but got %s", remoteCid, chunkCid)
			return cid.Undef
		}
	}

	return contentCid
}

func fetchKey(repo string) ic.PrivKey {
	kstorePath, err := homedir.Expand(filepath.Join(repo, "keystore"))
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
