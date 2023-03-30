package ali_driver

import (
	"github.com/go-resty/resty/v2"
	"time"
)

const (
	APITVPanBase = "https://api.tv-pan.com"
	APITVPanQR   = APITVPanBase + "/ali/qrcode"
	APITVPanAuth = APITVPanBase + "/ali/auth"

	APIBase              = "https://open.aliyundrive.com"
	APIList              = APIBase + "/adrive/v1.0/openFile/list"
	APIFile              = APIBase + "/adrive/v1.0/openFile/get"
	APIDriveInfo         = APIBase + "/adrive/v1.0/user/getDriveInfo"
	APIRemoveTrash       = APIBase + "/adrive/v1.0/openFile/recyclebin/trash"        //移动到垃圾箱
	APICreateFolder      = APIBase + "/adrive/v1.0/openFile/create"                  //创建目录
	APIFileDownload      = APIBase + "/adrive/v1.0/openFile/getDownloadUrl"          //获取下载链接
	APIFileVideoPlayInfo = APIBase + "/adrive/v1.0/openFile/getVideoPreviewPlayInfo" //获取视频转码播放信息
	APIFileMove          = APIBase + "/adrive/v1.0/openFile/move"                    //移动文件
	APIFileUpdate        = APIBase + "/adrive/v1.0/openFile/update"                  //更新文件
	APISpaceInfo         = APIBase + "/adrive/v1.0/user/getSpaceInfo"                //获取空间信息
)

var RestyHttpClient = NewRestyClient()
var UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36 Edg/110.0.1587.69"
var DefaultTimeout = time.Second * 30

func NewRestyClient() *resty.Client {
	return resty.New().
		SetHeader("user-agent", UserAgent).
		SetRetryCount(3).
		SetTimeout(DefaultTimeout)
}
