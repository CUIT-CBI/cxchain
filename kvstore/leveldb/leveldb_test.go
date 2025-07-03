package leveldb_test 

import (
    "testing"
    "chain/kvstore/leveldb"
)

func BenchmarkLevelDB_Put(b *testing.B) {
    db, err := leveldb.NewLevelDB("testdb_bench")
    if err != nil {
        b.Fatalf("OpenFile failed: %v", err)
    }
    defer db.Close()

    key := []byte("benchkey")
    val := []byte("benchvalue")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if err := db.Put(key, val); err != nil {
            b.Fatalf("Put failed: %v", err)
        }
    }
}

func BenchmarkLevelDB_Get(b *testing.B) {
    db, err := leveldb.NewLevelDB("testdb_bench")
    if err != nil {
        b.Fatalf("OpenFile failed: %v", err)
    }
    defer db.Close()

    key := []byte("benchkey")
    val := []byte("benchvalue")

    // 先写入数据
    if err := db.Put(key, val); err != nil {
        b.Fatalf("Put failed: %v", err)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if _, err := db.Get(key); err != nil {
            b.Fatalf("Get failed: %v", err)
        }
    }
}
