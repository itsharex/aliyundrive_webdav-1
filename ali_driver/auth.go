package ali_driver

import (
	"aliyundrive_webdav/db"
	"fmt"
	"github.com/fatih/color"
	"github.com/skip2/go-qrcode"
	"github.com/yanjunhui/aliyundrive_open"
	"time"
)

// GetQRCode  获取登录二维码信息
func GetQRCode() (result AliQrCodeResult, err error) {
	err = aliyundrive_open.HttpPost(APITVPanQR, nil, nil, &result)
	if err != nil {
		return result, err
	}

	if result.Error != "" {
		err = fmt.Errorf("获取二维码失败: %s", result.Error)
	}

	return result, err
}

// CheckQrCodeStatus 检查二维码状态. 通过 QRCode方法返回的 sid 参数获取二维码状态.
// 扫码成功后,返回 authCode 用于最后的登录授权获取 access_token 和 refresh_token
func CheckQrCodeStatus(sid string) (result AliQRCodeStatusResult, err error) {

	req := map[string]string{
		"sid":        sid,
		"grant_type": "qrcode_status",
	}

	err = aliyundrive_open.HttpPost(APITVPanAuth, nil, req, &result)
	if err != nil {
		return result, err
	}

	if result.Error != "" {
		err = fmt.Errorf("获取二维码失败: %s", result.Error)
	}

	return result, err
}

// Auth 登录授权.
// 通过 QrCodeStatus 方法返回的 authCode 参数获取 access_token 和 refresh_token
func Auth(authCode string) (authToken aliyundrive_open.Authorize, err error) {
	req := map[string]string{
		"authCode":   authCode,
		"grant_type": "authorization_code",
	}

	result := AliAuthResult{}
	err = aliyundrive_open.HttpPost(APITVPanAuth, nil, req, &result)
	if err != nil {
		return authToken, err
	}

	if result.Error != "" {
		err = fmt.Errorf("获取二维码失败: %s", result.Error)
	}

	authToken = result.Data

	return authToken, err
}

// RefreshToken 刷新 access_token
// 通过 Auth 方法返回的 refresh_token 参数刷新 access_token
func RefreshToken(refreshToken string) (authToken aliyundrive_open.Authorize, err error) {

	req := map[string]string{
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}

	result := AliAuthResult{}
	err = aliyundrive_open.HttpPost(APITVPanAuth, nil, req, &result)
	if err != nil {
		return authToken, err
	}

	if result.Error != "" {
		err = fmt.Errorf("获取二维码失败: %s", result.Error)
	}

	authToken = result.Data

	return authToken, db.SaveAccessToken(result.Data)
}

// 完整的登录授权流程
func LoginQRCode() (authToken aliyundrive_open.Authorize, err error) {

	qrCode, err := GetQRCode()
	if err != nil {
		return authToken, err
	}

	qrData := fmt.Sprintf("https://www.aliyundrive.com/o/oauth/authorize?sid=%s", qrCode.Data.Sid)
	obj, err := qrcode.New(qrData, qrcode.Low)
	if err == nil {
		color.Cyan("使用阿里云盘 App 扫码登录")
		fmt.Println(obj.ToSmallString(false))
	} else {
		color.Cyan("请打开以下网址扫码登陆: %s\n", qrCode.Data.QrCodeUrl)
	}

	authCode := ""
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			status, err := CheckQrCodeStatus(qrCode.Data.Sid)
			if err != nil {
				color.Cyan("获取二维码状态失败: %s\n", err)
				return authToken, err
			}
			switch status.Data.Status {
			case "WaitLogin":
				continue
			case "ScanSuccess":
				color.Cyan("%s\n", "二维码已扫描,等待授权确认")
			case "LoginSuccess":
				color.Cyan("%s\n", "二维码已确认")
				authCode = status.Data.AuthCode
			}
		}
		if authCode != "" {
			break
		}
	}

	//4. 登录授权
	authorize, err := Auth(authCode)
	if err != nil {
		color.Cyan("登录授权失败: %s\r", err)
		return authToken, err
	}

	return authorize, db.SaveAccessToken(authorize)
}
