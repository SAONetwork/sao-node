package types

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const PEER_INFO_PREFIX = "peerInfo_"
const FILE_INFO_PREFIX = "fileInfo_"

const CHUNK_SIZE int = 32 * 1024 * 1024

type PeerInfo struct {
	ID peer.ID
	//Agent       string
	Addrs []string
	//Protocols   []string
	//ConnMgrMeta *ConnMgrInfo
}

type ConnMgrInfo struct {
	FirstSeen time.Time
	Value     int
	Tags      map[string]int
	Conns     map[string]time.Time
}

type FileChunkReq struct {
	ChunkId     int
	TotalLength int
	TotalChunks int
	ChunkCid    string
	Cid         string
	Content     []byte
}

type ReceivedFileInfo struct {
	Cid            string
	TotalLength    int
	TotalChunks    int
	ReceivedLength int
	Path           string
	ChunkCids      []string
}

type RpcReq struct {
	Method string
	Params []string
}

type RpcResp struct {
	Data  string
	Error string
}
