package ali_driver

import (
	"aliyundrive_webdav/db"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/yanjunhui/aliyundrive_open"
	"log"
	"os"
	"strings"
	"time"
)

type File struct {
	aliyundrive_open.FileInfo
	ParentPath string `json:"parent_path"`
	Path       string `json:"path"`
}

var deBug = false

var FilesMapData = map[string][]File{}

func SaveFile() error {

	wo := &opt.WriteOptions{
		Sync: true,
	}

	save := func(file File) {
		key := makePathKey(file.Path)
		jsonData, err := json.Marshal(file)
		if err == nil {
			if deBug {
				err = writeJsonData(key, jsonData)
				if err != nil {
					log.Println("写入测试Json数据失败: ", err.Error())
					return
				}
			}

			err = db.DataBase.Client.Put([]byte(key), jsonData, wo)
			if err != nil {
				log.Printf("保存文件(%s)失败: %s", key, err.Error())
			}
		}
	}

	for key, value := range FilesMapData {
		//保存目录内文件信息
		for _, item := range value {
			if !item.IsDir() {
				save(item)
			}
		}

		//保存目录信息
		key = makePathKey(key)
		dirFile, err := GetListIndexData(key)
		if err == nil {
			err = SaveListIndexData(dirFile)
			if err != nil {
				log.Printf("保存文件(%s)列表索引信息失败: %s", key, err.Error())
			}
		}

		jsonData, err := json.Marshal(value)
		if err == nil {
			if deBug {
				err = writeJsonData(key, jsonData)
				if err != nil {
					log.Println("写入测试Json数据失败: ", err.Error())
					return err
				}
			}

			err = db.DataBase.Client.Put([]byte(key), jsonData, wo)
			if err != nil {
				log.Printf("保存文件(%s)列表失败: %s", key, err.Error())
			}
		}
	}

	return nil
}

func SaveListIndexData(file File) error {
	wo := &opt.WriteOptions{
		Sync: true,
	}

	file.UpdatedAt = time.Now()

	key := makePathKey(file.Path)
	key = fmt.Sprintf("index_%s", key)

	jsonData, err := json.Marshal(file)
	if err != nil {
		return err
	}
	if deBug {
		err = writeJsonData(key, jsonData)
		if err != nil {
			log.Println("写入测试Json数据失败: ", err.Error())
			return err
		}
	}

	return db.DataBase.Client.Put([]byte(key), jsonData, wo)
}

func GetListIndexData(path string) (file File, err error) {

	ro := &opt.ReadOptions{DontFillCache: false}

	key := makePathKey(path)
	key = fmt.Sprintf("index_%s", key)

	jsonData, err := db.DataBase.Client.Get([]byte(key), ro)
	if err == nil {
		err = json.Unmarshal(jsonData, &file)
	}

	return file, err
}

func RemoveListIndexData(path string) error {

	key := makePathKey(path)
	key = fmt.Sprintf("index_%s", key)

	return db.DataBase.DelData(key, true)
}

func GetFiles(path string) (list []File, err error) {
	ro := &opt.ReadOptions{DontFillCache: false}

	key := makePathKey(path)

	dataByte, err := db.DataBase.Client.Get([]byte(key), ro)
	if err == nil {
		err = json.Unmarshal(dataByte, &list)
	}

	return list, err
}

func UpdateFile(path string, file File) error {
	wo := &opt.WriteOptions{Sync: true}

	key := makePathKey(path)

	jsonData, err := json.Marshal(file)
	if err != nil {
		return err
	}

	return db.DataBase.Client.Put([]byte(key), jsonData, wo)
}

func RemoveFile(path string) error {
	wo := &opt.WriteOptions{Sync: false}
	key := makePathKey(path)

	return db.DataBase.Client.Delete([]byte(key), wo)
}

func GetPlayInfo(path string) (file File, err error) {

	ro := &opt.ReadOptions{DontFillCache: false}

	key := makePathKey(path)

	dataByte, err := db.DataBase.Client.Get([]byte(key), ro)
	if err != nil {
		RemoveFile(path)
		RemoveListIndexData(path)
		return file, err
	}

	err = json.Unmarshal(dataByte, &file)
	if err != nil {
		log.Printf("解析文件(%s)数据失败: %s", path, err.Error())
		RemoveFile(path)
		RemoveListIndexData(path)
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

func makePathKey(path string) string {
	path = strings.Replace(path, "/", "_", -1)
	driverID, err := GetDefaultDriveId()
	if err == nil {
		path = fmt.Sprintf("%s%s_%s", PrefixAli, driverID, path)
	}
	return path
}

func writeJsonData(name string, data []byte) error {
	path := "jsonData"
	_, err := os.Stat(path)
	if err != nil {
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	file, err := os.Create("jsonData/" + name + ".json")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
