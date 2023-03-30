package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/yanjunhui/aliyundrive_open"
)

const (
	PrefixAliToken     = "ali_"
	DefaultOpenDriveId = "defaultOpenDriveID"
)

func SaveDefaultOpenDriveId(driveId string) error {
	return DataBase.SetString(DefaultOpenDriveId, driveId, true)
}

func GetDefaultOpenDriveId() (driveId string, err error) {
	return DataBase.GetString(DefaultOpenDriveId, true)
}

func SaveAccessToken(token aliyundrive_open.Authorize) error {

	if token.AccessToken == "" {
		return errors.New("access_token is empty")
	}

	driveID, err := GetDefaultOpenDriveId()
	if err != nil || driveID == "" {
		err = SaveDefaultOpenDriveId(token.DriveID)
		if err != nil {
			return err
		}
	}

	key := fmt.Sprintf("%s%s", PrefixAliToken, token.DriveID)

	wo := &opt.WriteOptions{
		Sync: false,
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return DataBase.Client.Put([]byte(key), data, wo)
}

func RemoveDefaultAccessToken() error {
	driveID, err := GetDefaultOpenDriveId()
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", PrefixAliToken, driveID)
	return DataBase.DelData(key, true)
}

func RemoveAccessToken(driveId string) error {
	key := fmt.Sprintf("%s%s", PrefixAliToken, driveId)
	return DataBase.DelData(key, true)
}

func GetDefaultAccessToken() (accessToken aliyundrive_open.Authorize, err error) {
	driverID, err := GetDefaultOpenDriveId()
	if err != nil {
		return
	}

	return GetAccessToken(driverID)
}

func GetAccessToken(driveId string) (accessToken aliyundrive_open.Authorize, err error) {

	key := fmt.Sprintf("%s%s", PrefixAliToken, driveId)

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

func SaveWabDavUser(username, password string) error {
	return DataBase.SetString(username, password, false)
}

func GetWabDavUser(username string) (password string, err error) {
	return DataBase.GetString(username, true)
}
