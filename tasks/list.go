package tasks

import (
	"aliyundrive_webdav/ali_driver"
	"github.com/fatih/color"
	"os"
	"time"
)

func InitTasks(WorkPath string) {
	go func(path string) {
		ticker := time.NewTicker(time.Hour)
		for {
			select {
			case <-ticker.C:
				logFile, err := os.OpenFile(path+"/webdav.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 664)
				if err == nil {
					color.Output = logFile
				} else {
					color.Output = os.Stdout
				}

				ali_driver.ListAllFile()
			}
		}
	}(WorkPath)
}
