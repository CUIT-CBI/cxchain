package leveldb  

import (
    "github.com/syndtr/goleveldb/leveldb"  
    "chain/kvstore"  
)

type LevelDB struct {
    db *leveldb.DB
}

func NewLevelDB(path string) (kvstore.KVStore, error) {
	// 创建一个数据库实例
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	// 返回数据库实例
	return &LevelDB{db: db}, nil
}

func (l *LevelDB) Get(key []byte) ([]byte, error) {
    value, err := l.db.Get(key, nil)
    if err != nil {
        // 不要将 ErrNotFound 转换为 nil 错误
        // 原始代码: if err == leveldb.ErrNotFound { return nil, nil }
        return nil, err // 直接返回错误，包括 ErrNotFound
    }
    return value, nil
}

func (l *LevelDB) Put(key, value []byte) error {
	err := l.db.Put(key, value, nil)
	if err != nil {
		return err
	}
	return nil
}

func (l *LevelDB) Delete(key []byte) error {
	err := l.db.Delete(key, nil)
	if err != nil {
		return err
	}
	return nil
}

func (l *LevelDB) Has(key []byte) (bool, error) {
	_, err := l.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil // Key does not exist
		}
		return false, err // Other error
	}
	return true, nil // Key exists
}

func (l *LevelDB) Close() error {
	if err := l.db.Close(); err != nil {
		return err
	}
	return nil
}