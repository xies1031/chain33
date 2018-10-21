package mavl

import (
	lru "github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	mavl "gitlab.33.cn/chain33/chain33/system/store/mavl/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var mlog = log.New("module", "mavl")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

type Store struct {
	*drivers.BaseStore
	trees            map[string]*mavl.Tree
	cache            *lru.Cache
	enableMavlPrefix bool
	enableMVCC       bool
}

func init() {
	drivers.Reg("mavl", New)
}

type subConfig struct {
	EnableMavlPrefix bool `json:"enableMavlPrefix"`
	EnableMVCC       bool `json:"enableMVCC"`
}

func New(cfg *types.Store, sub []byte) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, subcfg)
	}
	mavls := &Store{bs, make(map[string]*mavl.Tree), nil, subcfg.EnableMavlPrefix, subcfg.EnableMVCC}
	mavls.cache, _ = lru.New(10)
	//使能前缀mavl以及MVCC

	mavls.enableMavlPrefix = subcfg.EnableMavlPrefix
	mavls.enableMVCC = subcfg.EnableMVCC
	mavl.EnableMavlPrefix(mavls.enableMavlPrefix)
	mavl.EnableMVCC(mavls.enableMVCC)
	bs.SetChild(mavls)
	return mavls
}

func (mavls *Store) Close() {
	mavls.BaseStore.Close()
	mlog.Info("store mavl closed")
}

func (mavls *Store) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	return mavl.SetKVPair(mavls.GetDB(), datas, sync)
}

func (mavls *Store) Get(datas *types.StoreGet) [][]byte {
	var tree *mavl.Tree
	var err error
	values := make([][]byte, len(datas.Keys))
	search := string(datas.StateHash)
	if data, ok := mavls.cache.Get(search); ok {
		tree = data.(*mavl.Tree)
	} else if data, ok := mavls.trees[search]; ok {
		tree = data
	} else {
		tree = mavl.NewTree(mavls.GetDB(), true)
		err = tree.Load(datas.StateHash)
		if err == nil {
			mavls.cache.Add(search, tree)
		}
		mlog.Debug("store mavl get tree", "err", err, "StateHash", common.ToHex(datas.StateHash))
	}
	if err == nil {
		for i := 0; i < len(datas.Keys); i++ {
			_, value, exit := tree.Get(datas.Keys[i])
			if exit {
				values[i] = value
			}
		}
	}
	return values
}

func (mavls *Store) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	if len(datas.KV) == 0 {
		mlog.Info("store mavl memset,use preStateHash as stateHash for kvset is null")
		mavls.trees[string(datas.StateHash)] = nil
		return datas.StateHash, nil
	}
	tree := mavl.NewTree(mavls.GetDB(), sync)
	err := tree.Load(datas.StateHash)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(datas.KV); i++ {
		tree.Set(datas.KV[i].Key, datas.KV[i].Value)
	}
	hash := tree.Hash()
	mavls.trees[string(hash)] = tree
	if len(mavls.trees) > 1000 {
		mlog.Error("too many trees in cache")
	}
	return hash, nil
}

func (mavls *Store) Commit(req *types.ReqHash) ([]byte, error) {
	tree, ok := mavls.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}

	if tree == nil {
		mlog.Info("store mavl commit,do nothing for kvset is null")
		delete(mavls.trees, string(req.Hash))
		return req.Hash, nil
	}

	hash := tree.Save()
	if hash == nil {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrDataBaseDamage
	}
	delete(mavls.trees, string(req.Hash))
	return req.Hash, nil
}

func (mavls *Store) Rollback(req *types.ReqHash) ([]byte, error) {
	_, ok := mavls.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mavl rollback", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	delete(mavls.trees, string(req.Hash))
	return req.Hash, nil
}

func (mavls *Store) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	mavl.IterateRangeByStateHash(mavls.GetDB(), statehash, start, end, ascending, fn)
}

func (mavls *Store) ProcEvent(msg queue.Message) {
	msg.ReplyErr("Store", types.ErrActionNotSupport)
}

func (mavls *Store) Del(req *types.StoreDel) ([]byte, error) {
	//not support
	return nil, nil
}
