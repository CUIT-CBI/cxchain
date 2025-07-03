package leveldb_test

import (
	"os"
	"testing"
	"chain/kvstore"
	"chain/kvstore/leveldb"
)

func TestLevelDB(t *testing.T) {
	path := "testdb"
	defer os.RemoveAll(path) // 测试结束后清理数据库文件夹

	var db kvstore.KVStore
	var err error

	// 创建 LevelDB 实例
	db, err = leveldb.NewLevelDB(path)
	if err != nil {
		t.Fatalf("Failed to open LevelDB: %v", err)
	}
	defer db.Close()

	// 测试 Put 操作
	key := []byte("testkey")
	value := []byte("testvalue")
	if err := db.Put(key, value); err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	// 测试 Get 操作
	retrievedValue, err := db.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if string(retrievedValue) != string(value) {
		t.Fatalf("Expected value %s, got %s", value, retrievedValue)
	}

	// 测试 Has 操作
	has, err := db.Has(key)
	if err != nil {
		t.Fatalf("Failed to check if key exists: %v", err)
	}
	if !has {
		t.Fatalf("Expected key to exist, but it does not")
	}

	// 测试 Delete 操作
	if err := db.Delete(key); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// 确认删除后无法获取 key
	_, err = db.Get(key)
	if err == nil {
		t.Fatalf("Expected key to be deleted, but it still exists")
	}
}
