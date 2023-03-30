package db

import (
	"os"
	"strings"
)

var DataBase *FileDB

func InitDB(path string) (err error) {
	path = strings.Replace(path, "file://", "", 1)
	path = path + "/data"
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	DataBase = new(FileDB)
	DataBase, err = NewFileDB(path)
	return err
}
