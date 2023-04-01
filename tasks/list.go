package tasks

import (
	"aliyundrive_webdav/ali_driver"
	"time"
)

func InitTasks(WorkPath string) {
	go func(path string) {
		ticker := time.NewTicker(time.Hour)
		for {
			select {
			case <-ticker.C:
				ali_driver.ListAllFile()
			}
		}
	}(WorkPath)
}
