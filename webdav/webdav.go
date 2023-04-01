package webdav

import (
	"aliyundrive_webdav/ali_driver"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

func ServeWebDAV() {
	go func() {
		e := echo.New()
		e.Any("/*", ServeHTTP)

		err := e.Start(":6666")
		if err != nil {
			log.Print("webdav 启动失败: ", err)
		}
	}()
}

func ServeHTTP(ctx echo.Context) error {
	switch ctx.Request().Method {
	case "OPTIONS":
		handleOptions(ctx)
	case "GET", "HEAD", "POST":
		handleGetHeadPost(ctx)
	case "PROPFIND":
		handlePropfind(ctx)
	case "DELETE", "PUT", "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", "PROPPATCH":
		return ctx.NoContent(http.StatusMethodNotAllowed)
	}

	return ctx.NoContent(http.StatusMethodNotAllowed)
}

func handleOptions(ctx echo.Context) error {
	allow := "OPTIONS, PROPFIND, GET, HEAD, POST"
	ctx.Response().Header().Set("Allow", allow)
	// http://www.webdav.org/specs/rfc4918.html#dav.compliance.classes
	ctx.Response().Header().Set("DAV", "1, 2")
	// http://msdn.microsoft.com/en-au/library/cc250217.aspx
	ctx.Response().Header().Set("MS-Author-Via", "DAV")
	return ctx.NoContent(http.StatusOK)
}

// 获取文件链接
func handleGetHeadPost(ctx echo.Context) error {
	reqPath := ctx.Request().URL.Path

	if s := strings.TrimPrefix(reqPath, "/root/"); s == reqPath {
		reqPath = "root" + reqPath
	}

	reqPath = strings.TrimLeft(reqPath, "/")

	file, err := ali_driver.GetPlayInfo(reqPath)
	if err != nil {
		log.Printf("查询路径(%s) 失败: %s", reqPath, err.Error())
		return ctx.NoContent(http.StatusNotFound)
	}

	etag, err := findETag(file)
	if err != nil {
		return ctx.NoContent(http.StatusInternalServerError)
	}

	ctx.Response().Header().Set("ETag", etag)

	return ctx.Redirect(http.StatusFound, file.DownloadUrl)
}

// 获取目录
func handlePropfind(ctx echo.Context) error {
	reqPath := ctx.Request().URL.Path
	if reqPath == "/" {
		reqPath = "root"
	} else {
		reqPath = strings.TrimPrefix(reqPath, "/")
	}

	if !strings.HasPrefix(reqPath, "root/") {
		if reqPath != "root" {
			reqPath = "root/" + reqPath
		}
	}

	reqPath = strings.TrimRight(reqPath, "/")

	depth := infiniteDepth
	if hdr := ctx.Request().Header.Get("Depth"); hdr != "" {
		depth = parseDepth(hdr)
		if depth == invalidDepth {
			return ctx.NoContent(http.StatusBadRequest)
		}
	}

	file, err := ali_driver.GetListIndexData(reqPath)
	if err != nil {
		log.Printf("查询路径(%s) 失败: %s", reqPath, err.Error())
		return ctx.NoContent(http.StatusNotFound)
	}

	pf, status, err := readPropfind(ctx.Request().Body)
	if err != nil {
		return ctx.NoContent(status)
	}

	mw := multistatusWriter{w: ctx.Response()}
	walkFn := func(reqPath string, info ali_driver.File, err error) error {
		if err != nil {
			return err
		}
		var pstats []Propstat
		if pf.Propname != nil {
			pnames, err := propnames(info)
			if err != nil {
				return err
			}
			pstat := Propstat{Status: http.StatusOK}
			for _, xmlname := range pnames {
				pstat.Props = append(pstat.Props, Property{XMLName: xmlname})
			}
			pstats = append(pstats, pstat)
		} else if pf.Allprop != nil {
			pstats, err = allprop(info, pf.Prop)
		} else {
			pstats, err = props(info, pf.Prop)
		}
		if err != nil {
			return err
		}

		href := reqPath
		if info.IsDir() {
			href = strings.TrimLeft(href, "root")
			href += "/"
		}

		return mw.write(makePropstatResponse(href, pstats))
	}

	walkErr := walkFS(reqPath, file, depth, walkFn)
	closeErr := mw.close()
	if walkErr != nil {
		return ctx.NoContent(http.StatusInternalServerError)
	}
	if closeErr != nil {
		return ctx.NoContent(http.StatusInternalServerError)
	}
	return ctx.NoContent(http.StatusNotFound)
}

// walkFS traverses filesystem fs starting at name up to depth levels.
//
// Allowed values for depth are 0, 1 or infiniteDepth. For each visited node,
// walkFS calls walkFn. If a visited file system node is a directory and
// walkFn returns filepath.SkipDir, walkFS will skip traversal of this node.
func walkFS(path string, fi ali_driver.File, depth int, walkFn func(reqPath string, info ali_driver.File, err error) error) error {
	// This implementation is based on Walk's code in the standard path/filepath package.
	err := walkFn(path, fi, nil)
	if err != nil {
		if fi.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !fi.IsDir() || depth == 0 {
		return nil
	}

	if depth == 1 {
		depth = 0
	}

	list, err := ali_driver.GetFiles(path)
	if err != nil {
		return err
	}

	for _, file := range list {
		file.Path = path + "/" + file.Name
		file.ParentPath = path
		err = walkFS(file.Path, file, depth, walkFn)
		if err != nil {
			if !file.IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}

	return nil
}

func makePropstatResponse(href string, pstats []Propstat) *response {
	resp := response{
		Href:     []string{(&url.URL{Path: href}).EscapedPath()},
		Propstat: make([]propstat, 0, len(pstats)),
	}
	for _, p := range pstats {
		var xmlErr *xmlError
		if p.XMLError != "" {
			xmlErr = &xmlError{InnerXML: []byte(p.XMLError)}
		}
		resp.Propstat = append(resp.Propstat, propstat{
			Status:              fmt.Sprintf("HTTP/1.1 %d %s", p.Status, StatusText(p.Status)),
			Prop:                p.Props,
			ResponseDescription: p.ResponseDescription,
			Error:               xmlErr,
		})
	}
	return &resp
}

const (
	infiniteDepth = -1
	invalidDepth  = -2
)

// parseDepth maps the strings "0", "1" and "infinity" to 0, 1 and
// infiniteDepth. Parsing any other string returns invalidDepth.
//
// Different WebDAV methods have further constraints on valid depths:
//   - PROPFIND has no further restrictions, as per section 9.1.
//   - COPY accepts only "0" or "infinity", as per section 9.8.3.
//   - MOVE accepts only "infinity", as per section 9.9.2.
//   - LOCK accepts only "0" or "infinity", as per section 9.10.3.
//
// These constraints are enforced by the handleXxx methods.
func parseDepth(s string) int {
	switch s {
	case "0":
		return 0
	case "1":
		return 1
	case "infinity":
		return infiniteDepth
	}
	return invalidDepth
}

// http://www.webdav.org/specs/rfc4918.html#status.code.extensions.to.http11
const (
	StatusMulti               = 207
	StatusUnprocessableEntity = 422
	StatusLocked              = 423
	StatusFailedDependency    = 424
	StatusInsufficientStorage = 507
)

func StatusText(code int) string {
	switch code {
	case StatusMulti:
		return "Multi-Status"
	case StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case StatusLocked:
		return "Locked"
	case StatusFailedDependency:
		return "Failed Dependency"
	case StatusInsufficientStorage:
		return "Insufficient Storage"
	}
	return http.StatusText(code)
}

var (
	errDestinationEqualsSource = errors.New("webdav: destination equals source")
	errDirectoryNotEmpty       = errors.New("webdav: directory not empty")
	errInvalidDepth            = errors.New("webdav: invalid depth")
	errInvalidDestination      = errors.New("webdav: invalid destination")
	errInvalidIfHeader         = errors.New("webdav: invalid If header")
	errInvalidLockInfo         = errors.New("webdav: invalid lock info")
	errInvalidLockToken        = errors.New("webdav: invalid lock token")
	errInvalidPropfind         = errors.New("webdav: invalid propfind")
	errInvalidProppatch        = errors.New("webdav: invalid proppatch")
	errInvalidResponse         = errors.New("webdav: invalid response")
	errInvalidTimeout          = errors.New("webdav: invalid timeout")
	errNoFileSystem            = errors.New("webdav: no file system")
	errNoLockSystem            = errors.New("webdav: no lock system")
	errNotADirectory           = errors.New("webdav: not a directory")
	errPrefixMismatch          = errors.New("webdav: prefix mismatch")
	errRecursionTooDeep        = errors.New("webdav: recursion too deep")
	errUnsupportedLockInfo     = errors.New("webdav: unsupported lock info")
	errUnsupportedMethod       = errors.New("webdav: unsupported method")
)
