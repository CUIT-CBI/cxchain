package kvstore

import "github.com/syndtr/goleveldb/leveldb"

type LevelDB struct {
	db *leveldb.DB
}

func NewLevelDB(path string) *LevelDB {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		panic(err)
	}
	return &LevelDB{
		db: db,
	}
}

func (ldb *LevelDB) Put(key, value []byte) error {
	return ldb.db.Put(key, value, nil)
}

func (ldb *LevelDB) Get(key []byte) ([]byte, error) {
	return ldb.db.Get(key, nil)
}

func (ldb *LevelDB) Exist(key []byte) (bool, error) {
	return ldb.db.Has(key, nil)
}

func (ldb *LevelDB) Delete(key []byte) error {
	return ldb.db.Delete(key, nil)
}

func (ldb *LevelDB) Close() error {
	return ldb.db.Close()
}
