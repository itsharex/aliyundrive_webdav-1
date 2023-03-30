package ali_driver

import "github.com/yanjunhui/aliyundrive_open"

type ResultError struct {
	Code  int    `json:"code"`
	Error string `json:"error,omitempty"`
}

type AliQrCodeResult struct {
	Data aliyundrive_open.AuthorizeQRCode `json:"data"`
	ResultError
}

type AliQRCodeStatusResult struct {
	Data aliyundrive_open.AuthorizeQRCodeStatus `json:"data"`
	ResultError
}

type AliAuthResult struct {
	Data aliyundrive_open.Authorize `json:"data"`
	ResultError
}
