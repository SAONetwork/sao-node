package repo

import (
	"os"
	"path/filepath"

	"github.com/SaoNetwork/sao-node/types"

	dgbadger "github.com/dgraph-io/badger/v2"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	levelds "github.com/ipfs/go-ds-leveldb"
	measure "github.com/ipfs/go-ds-measure"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	dsNsMetadata  = "metadata"
	dsNsOrder     = "order"
	dsNsTransport = "transport"
	dsNsIndexer   = "indexer"
)

type dsCtor func(path string, readonly bool) (datastore.Batching, error)

var fsDatastores = map[string]dsCtor{
	dsNsMetadata: levelDs,
	// Those need to be fast for large writes... but also need a really good GC
	dsNsOrder:     badgerDs,
	dsNsTransport: levelDs,
	dsNsIndexer:   levelDs,
}

func levelDs(path string, readonly bool) (datastore.Batching, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func badgerDs(path string, readonly bool) (datastore.Batching, error) {
	opts := badger.DefaultOptions
	opts.ReadOnly = readonly

	opts.Options = dgbadger.DefaultOptions("").WithTruncate(true).
		WithValueThreshold(1 << 10)
	return badger.NewDatastore(path, &opts)
}

func (fsr *Repo) openDatastores(readonly bool) (map[string]datastore.Batching, error) {
	if err := os.MkdirAll(fsr.join(fsDatastore), 0755); err != nil {
		return nil, types.Wrapf(types.ErrCreateDirFailed, "mkdir %s: %v", fsr.join(fsDatastore), err)
	}

	out := map[string]datastore.Batching{}

	for p, ctor := range fsDatastores {
		// TODO: optimization: don't init datastores we don't need
		ds, err := ctor(fsr.join(filepath.Join(fsDatastore, p)), readonly)
		if err != nil {
			return nil, types.Wrap(types.ErrOpenDataStoreFailed, err)
		}

		ds = measure.New("fsrepo."+p, ds)

		out[datastore.NewKey(p).String()] = ds
	}

	return out, nil
}
