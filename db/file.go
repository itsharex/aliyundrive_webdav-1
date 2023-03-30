package db

import (
	"encoding/base64"
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/yanjunhui/aliyundrive_open"
	"log"
	"strings"
)

type File struct {
	aliyundrive_open.FileInfo
	ParentPath string `json:"parent_path"`
	Path       string `json:"path"`
}

var FilesMapData = map[string][]File{}

func SaveFile() error {
	wo := &opt.WriteOptions{
		Sync: false,
	}
	for key, value := range FilesMapData {
		for _, item := range value {
			if !item.IsDir() {
				fileKey := strings.Replace(item.Path, "/", "_", -1)
				fileData, err := json.Marshal(item)
				if err == nil {
					err = DataBase.Client.Put([]byte(fileKey), fileData, wo)
					if err != nil {
						log.Printf("保存文件(%s)失败: %s", key, err.Error())
					}
				}
			}
		}

		key = strings.Replace(key, "/", "_", -1)
		dirFile, err := GetListIndexData(key)
		if err == nil {
			err = SaveListIndexData(dirFile)
			if err != nil {
				log.Printf("保存文件(%s)列表索引信息失败: %s", key, err.Error())
			}
		}

		jsonData, err := json.Marshal(value)
		if err == nil {
			err = DataBase.Client.Put([]byte(key), jsonData, wo)
			if err != nil {
				log.Printf("保存文件(%s)列表失败: %s", key, err.Error())
			}
		}
	}

	return nil
}

func SaveListIndexData(file File) error {
	wo := &opt.WriteOptions{
		Sync: false,
	}

	key := strings.Replace(file.Path, "/", "_", -1)
	key = "index_" + key

	jsonData, err := json.Marshal(file)
	if err != nil {
		return err
	}

	return DataBase.Client.Put([]byte(key), jsonData, wo)
}

func GetListIndexData(path string) (file File, err error) {
	ro := &opt.ReadOptions{DontFillCache: false}
	key := strings.Replace(path, "/", "_", -1)
	key = "index_" + key

	jsonData, err := DataBase.Client.Get([]byte(key), ro)
	if err == nil {
		err = json.Unmarshal(jsonData, &file)
	}

	return file, err
}

func GetFile(path string) (file File, err error) {
	ro := &opt.ReadOptions{DontFillCache: false}
	key := strings.Replace(path, "/", "_", -1)
	dataByte, err := DataBase.Client.Get([]byte(key), ro)
	if err == nil {
		err = json.Unmarshal(dataByte, &file)
		if err == nil {
			authToken, err := GetDefaultAccessToken()
			if err == nil {
				option := aliyundrive_open.NewFileOption(authToken.DriveID, authToken.AccessToken)
				file.FileInfo, err = authToken.File(option)
			}
		}
	}

	return file, err
}

func GetFiles(path string) (list []File, err error) {
	ro := &opt.ReadOptions{DontFillCache: false}

	key := strings.Replace(path, "/", "_", -1)

	dataByte, err := DataBase.Client.Get([]byte(key), ro)
	if err == nil {
		err = json.Unmarshal(dataByte, &list)
	}

	return list, err
}

func UpdateFile(path string, file File) error {
	wo := &opt.WriteOptions{Sync: false}
	key := strings.Replace(path, "/", "_", -1)
	jsonData, err := json.Marshal(file)
	if err != nil {
		return err
	}

	return DataBase.Client.Put([]byte(key), jsonData, wo)
}

func RemoveFile(path string) error {
	wo := &opt.WriteOptions{Sync: false}
	key := strings.Replace(path, "/", "_", -1)

	return DataBase.Client.Delete([]byte(key), wo)
}

func GetPlayInfo(path string) (file File, err error) {

	ro := &opt.ReadOptions{DontFillCache: false}
	key := strings.Replace(path, "/", "_", -1)

	dataByte, err := DataBase.Client.Get([]byte(key), ro)
	if err != nil {
		return file, err
	}

	err = json.Unmarshal(dataByte, &file)
	if err != nil {
		log.Printf("解析文件(%s)数据失败: %s", path, err.Error())
		RemoveFile(path)
		return file, err
	}

	authToken, err := GetDefaultAccessToken()
	if err != nil {
		return file, err
	}

	option := aliyundrive_open.NewFileDownloadURLOption(authToken.DriveID, file.FileId)
	downInfo, err := authToken.FileDownloadURL(option)
	if err != nil {
		log.Printf("获取文件(%s)下载信息失败: %s", path, err.Error())
		return file, err
	}

	file.DownloadUrl = downInfo.URL

	//避免存储时特殊符号导致的错误
	bu := URLStringToBase64(file.DownloadUrl)
	if bu != "" {
		file.DownloadUrl = bu
	}

	//缓存数据
	err = UpdateFile(path, file)
	if err != nil {
		log.Printf("更新文件(%s)下载信息失败: %s", path, err.Error())
	}

	//缓存完后还原数据
	file.DownloadUrl = URLBase64ToString(file.DownloadUrl)

	return file, err
}

func URLStringToBase64(downloadUrl string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(downloadUrl))
}

func URLBase64ToString(downloadUrl string) string {
	u, err := base64.RawURLEncoding.DecodeString(downloadUrl)
	if err == nil {
		return string(u)
	}
	return ""
}
