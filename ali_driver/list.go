package ali_driver

import (
	"github.com/fatih/color"
	"github.com/yanjunhui/aliyundrive_open"
	"strings"
)

var PrintLog = false

func ListAllFile() error {
	FilesMapData = make(map[string][]File)
	err := GetFileList(File{}, "")
	if err == nil {
		return SaveFile()
	}
	return err
}

func GetFileList(file File, nextMarker string) error {
	authToken, err := GetDefaultAccessToken()
	if err != nil {
		return err
	}

	file.DriveId = authToken.DriveID

	if file.Path == "" {
		file.Path = "root"
		file.Type = "folder"
	} else {
		path := file.Path
		option := aliyundrive_open.NewFileOption(authToken.DriveID, file.FileId)
		nFile, err := authToken.File(option)
		if err != nil {
			return err
		}
		file.FileInfo = nFile
		file.Path = path
	}

	if nextMarker == "" {
		nextMarker = "next"
	}

	//存储目录信息
	err = SaveListIndexData(file)
	if err != nil {
		return err
	}

	for nextMarker != "" {
		if nextMarker == "next" {
			nextMarker = ""
		}

		if file.FileId == "" {
			file.FileId = "root"
		}

		option := aliyundrive_open.NewFileListOption(authToken.DriveID, file.FileId, nextMarker)
		list, err := authToken.FileList(option)
		if err != nil {
			return err
		}

		for _, item := range list.Items {
			nItem := File{}
			nItem.FileInfo = item
			nItem.ParentPath = file.Path
			nItem.Path = file.Path + "/" + item.Name
			key := strings.Replace(nItem.ParentPath, "/", "_", -1)
			FilesMapData[key] = append(FilesMapData[key], nItem)
			if item.IsDir() {
				if PrintLog {
					color.Cyan("扫描目录(%s)内文件", item.Name)
				}
				GetFileList(nItem, "")
			} else {
				if PrintLog {
					color.Green("保存文件(%s)成功", item.Name)
				}
			}
		}

		nextMarker = list.NextMarker
	}

	return nil
}
