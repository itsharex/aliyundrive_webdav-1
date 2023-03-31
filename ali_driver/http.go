package ali_driver

import (
	"github.com/go-resty/resty/v2"
	"time"
)

const (
	APITVPanBase = "https://api.tv-pan.com"
	APITVPanQR   = APITVPanBase + "/ali/qrcode"
	APITVPanAuth = APITVPanBase + "/ali/auth"
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
