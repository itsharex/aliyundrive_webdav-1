package main

import (
	"aliyundrive_webdav/ali_driver"
	"aliyundrive_webdav/db"
	"aliyundrive_webdav/webdav"
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var WorkPath = getCurrentAbPath()

func init() {
	db.InitDB(WorkPath)
}

func main() {
	args := os.Args
	if len(args) > 1 {
		switch strings.ToLower(args[1]) {
		case "start":
			_, err := db.GetDefaultAccessToken()
			if err != nil {
				fmt.Println("未登录阿里云盘, 请执行 login 登录")
				return
			} else {
				StartServer()
			}
		case "stop":
			StopServer()
		case "status":
			pid, status := GetRunStatus()
			if status {
				color.Green("服务正在运行, PID: %d", pid)
			} else {
				color.Red("服务未运行")
			}
		case "login":
			authToken, err := ali_driver.LoginQRCode()
			if err == nil {
				db.SaveAccessToken(authToken.Data)
				info, err := authToken.Data.DriveInfo()
				if err == nil {
					space, err := authToken.Data.DriveSpace()
					if err == nil {
						fmt.Printf("云盘ID: %s\n总容量: %d GB\n已用: %d GB\n可用: %d GB\n",
							info.DefaultDriveId,
							space.PersonalSpaceInfo.TotalSize/1024/1024/1024, space.PersonalSpaceInfo.UsedSize/1024/1024/1024,
							(space.PersonalSpaceInfo.TotalSize-space.PersonalSpaceInfo.UsedSize)/1024/1024/1024)
						color.Red("%s\n", "登录成功, 可以执行 \"get\" 参数获取文件列表啦 ")
						return
					}
				}
			}
			color.Red("登录失败, 请重试!")
		case "get":
			_, ok := GetRunStatus()
			if ok {
				StopServer()
				color.Red("Webdav 已停止, 请采集完成后再手动启动服务")
			}
			authToken, err := db.GetDefaultAccessToken()
			if err == nil {
				if authToken.ExpiresTime.After(time.Now()) {
					authTokenR, err := ali_driver.RefreshToken(authToken.RefreshToken)
					if err == nil {
						db.SaveAccessToken(authTokenR.Data)
						err = ali_driver.GetFileList(db.File{}, "")
						if err == nil {
							err = db.SaveFile()
							if err == nil {
								color.Red("%s\n", "获取云盘目录成功, 可以执行 \"start\" 参数启动 Webdav 服务啦")
								return
							}
						}
					}
				}
			}
			color.Red("%s\r", "获取云盘目录失败, 可能是未登录或授权已失效, 请重新登录")
		default:
			PrintHelp()
		}
	} else {
		PrintHelp()
	}

}

func PrintHelp() {
	fmt.Println(`
Start	启动 Webdav 

Stop  	停止 Webdav

Status 	查看 Webdav

Login	登录阿里云盘

Get	获取云盘目录`)
}

func StopServer() {
	fmt.Println("正在停止服务...")
	pid, status := GetRunStatus()
	if status {
		ps, err := os.FindProcess(pid)
		if err == nil {
			err = ps.Kill()
			if err == nil {
				return
			}
		}
		color.Red("服务终止失败, 请手动终止进程: ", pid)
	} else {
		color.Red("服务未运行")
	}
	pidFile, err := os.Stat(path.Join(WorkPath, "pid"))
	if err == nil {
		err := os.Remove(pidFile.Name())
		if err != nil {
			fmt.Println("删除pid文件失败: ", err)
		}
	}
}

func StartServer() {
	pid, status := GetRunStatus()
	if status {
		color.Green("程序已经在运行中, PID: ", pid)
		return
	}

	var e *echo.Echo
	e = echo.New()
	e.Any("/*", webdav.ServeHTTP)
	err := GracefulHttp(e, strings.Join([]string{"0.0.0.0", fmt.Sprint(6969)}, ":"))
	if err != nil {
		fmt.Println("服务启动失败: ", err)
	}
}

// GetRunStatus 获取进程状态
func GetRunStatus() (pid int, status bool) {
	pidByte, err := os.ReadFile(path.Join(WorkPath, "pid"))
	if err != nil {
		return 0, false
	}

	pid, err = strconv.Atoi(string(pidByte))
	if err != nil {
		return 0, false
	}

	status, err = process.PidExists(int32(pid))
	if err != nil {
		return 0, false
	}

	return pid, status
}

func GracefulHttp(e *echo.Echo, address string) error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func(context context.Context) {
		for s := range c {
			switch s {
			case syscall.SIGHUP:
			case syscall.SIGINT:
				os.Exit(0)
			case syscall.SIGKILL:
				err := e.Shutdown(ctx)
				if err == nil {
					os.Exit(9)
				}
			}
		}
	}(ctx)

	err := os.WriteFile(path.Join(WorkPath, "pid"), []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	if err != nil {
		log.Errorf("写入pid文件失败: %s", err)
		return err
	}
	return e.Start(address)
}

func getCurrentAbPath() string {
	dir := getCurrentAbPathByExecutable()
	tmpDir, err := filepath.EvalSymlinks(os.TempDir())
	if err == nil && strings.Contains(dir, tmpDir) {
		if strings.Contains(dir, tmpDir) {
			return getCurrentAbPathByCaller()
		}
	}
	return dir
}

// 获取当前执行文件绝对路径
func getCurrentAbPathByExecutable() string {
	exePath, err := os.Executable()
	if err == nil {
		res, err := filepath.EvalSymlinks(filepath.Dir(exePath))
		if err == nil {
			return res
		}
	}
	return "./"
}

// 获取当前执行文件绝对路径（go run）
func getCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath = path.Dir(filename)
	}
	return abPath
}
