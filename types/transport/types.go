package transport

const PEER_INFO_PREFIX = "peerInfo_"
const FILE_INFO_PREFIX = "fileInfo_"

const CHUNK_SIZE int = 32 * 1024 * 1024

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
