package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type FileDB struct {
	Client *leveldb.DB
}

func NewFileDB(path string) (db *FileDB, err error) {
	db = new(FileDB)
	db.Client, err = leveldb.OpenFile(path, nil)
	return db, err
}

// 关闭数据库文件句柄
func (db *FileDB) Close() error {
	return db.Client.Close()
}

func (db *FileDB) SetString(key, value string, syn bool) error {
	wo := &opt.WriteOptions{
		Sync: syn,
	}

	return db.Client.Put([]byte(key), []byte(value), wo)
}

func (db *FileDB) GetString(key string, cache bool) (string, error) {
	ro := &opt.ReadOptions{
		DontFillCache: !cache,
	}

	data, err := db.Client.Get([]byte(key), ro)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (db *FileDB) DelData(key string, syn bool) error {
	wo := &opt.WriteOptions{
		Sync: syn,
	}

	return db.Client.Delete([]byte(key), wo)
}

func (db *FileDB) SetBool(key string, value bool, syn bool) error {
	wo := &opt.WriteOptions{
		Sync: syn,
	}

	var b = []byte{0}
	if value {
		b = []byte{1}
	}
	return db.Client.Put([]byte(key), b, wo)
}

func (db *FileDB) GetBool(key string, cache bool) (bool, error) {
	ro := &opt.ReadOptions{
		DontFillCache: !cache,
	}

	data, err := db.Client.Get([]byte(key), ro)
	if err != nil || len(data) == 0 {
		return false, err
	}
	return data[0] == 1, nil
}
