package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/yanjunhui/aliyundrive_open"
)

const (
	PrefixAli          = "ali_"
	DefaultOpenDriveId = "defaultOpenDriveID"
)

func SaveDefaultDriveId(driveId string) error {
	return DataBase.SetString(DefaultOpenDriveId, driveId, true)
}

func GetDefaultDriveId() (driveId string, err error) {
	return DataBase.GetString(DefaultOpenDriveId, true)
}

func SaveAccessToken(token aliyundrive_open.Authorize) error {
	if token.AccessToken == "" {
		return errors.New("access_token is empty")
	}

	err := SaveDefaultDriveId(token.DriveID)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s%s", PrefixAli, token.DriveID)

	wo := &opt.WriteOptions{
		Sync: false,
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return DataBase.Client.Put([]byte(key), data, wo)
}

func GetDefaultAccessToken() (accessToken aliyundrive_open.Authorize, err error) {
	driverID, err := GetDefaultDriveId()
	if err != nil {
		return
	}

	return GetAccessToken(driverID)
}

func GetAccessToken(driveId string) (accessToken aliyundrive_open.Authorize, err error) {

	key := fmt.Sprintf("%s%s", PrefixAli, driveId)

	ro := &opt.ReadOptions{
		DontFillCache: false,
	}

	dataByte, err := DataBase.Client.Get([]byte(key), ro)
	if err != nil {
		return accessToken, err
	}

	err = json.Unmarshal(dataByte, &accessToken)
	if err != nil {
		return accessToken, err
	}

	return accessToken, err
}
