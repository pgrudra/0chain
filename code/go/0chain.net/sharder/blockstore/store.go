package blockstore

import (
	"0chain.net/chaincore/block"
)

/*BlockStoreI - an interface to read and write blocks to some storage */
type BlockStoreI interface {
	Write(b *block.Block) error
	Read(hash string, round int64) (*block.Block, error)
	ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error)
}

var Store BlockStoreI

/*GetStore - get the block store that's is setup */
func GetStore() BlockStoreI {
	return Store
}

/*SetupStore - Setup a file system based block storage */
func SetupStore(store BlockStoreI) {
	Store = store
}
