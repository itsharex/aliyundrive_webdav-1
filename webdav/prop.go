// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdav

import (
	"aliyundrive_webdav/ali_driver"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
)

// Proppatch describes a property update instruction as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPPATCH
type Proppatch struct {
	// Remove specifies whether this patch removes properties. If it does not
	// remove them, it sets them.
	Remove bool
	// Props contains the properties to be set or removed.
	Props []Property
}

// Propstat describes a XML propstat element as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat
type Propstat struct {
	// Props contains the properties for which Status applies.
	Props []Property

	// Status defines the HTTP status code of the properties in Prop.
	// Allowed values include, but are not limited to the WebDAV status
	// code extensions for HTTP/1.1.
	// http://www.webdav.org/specs/rfc4918.html#status.code.extensions.to.http11
	Status int

	// XMLError contains the XML representation of the optional error element.
	// XML content within this field must not rely on any predefined
	// namespace declarations or prefixes. If empty, the XML error element
	// is omitted.
	XMLError string

	// ResponseDescription contains the contents of the optional
	// responsedescription field. If empty, the XML element is omitted.
	ResponseDescription string
}

// makePropstats returns a slice containing those of x and y whose Props slice
// is non-empty. If both are empty, it returns a slice containing an otherwise
// zero Propstat whose HTTP status code is 200 OK.
func makePropstats(x, y Propstat) []Propstat {
	pstats := make([]Propstat, 0, 2)
	if len(x.Props) != 0 {
		pstats = append(pstats, x)
	}
	if len(y.Props) != 0 {
		pstats = append(pstats, y)
	}
	if len(pstats) == 0 {
		pstats = append(pstats, Propstat{
			Status: http.StatusOK,
		})
	}
	return pstats
}

// DeadPropsHolder holds the dead properties of a resource.
//
// Dead properties are those properties that are explicitly defined. In
// comparison, live properties, such as DAV:getcontentlength, are implicitly
// defined by the underlying resource, and cannot be explicitly overridden or
// removed. See the Terminology section of
// http://www.webdav.org/specs/rfc4918.html#rfc.section.3
//
// There is a whitelist of the names of live properties. This package handles
// all live properties, and will only pass non-whitelisted names to the Patch
// method of DeadPropsHolder implementations.
type DeadPropsHolder interface {
	// DeadProps returns a copy of the dead properties held.
	DeadProps() (map[xml.Name]Property, error)

	// Patch patches the dead properties held.
	//
	// Patching is atomic; either all or no patches succeed. It returns (nil,
	// non-nil) if an internal server error occurred, otherwise the Propstats
	// collectively contain one Property for each proposed patch Property. If
	// all patches succeed, Patch returns a slice of length one and a Propstat
	// element with a 200 OK HTTP status code. If none succeed, for reasons
	// other than an internal server error, no Propstat has status 200 OK.
	//
	// For more details on when various HTTP status codes apply, see
	// http://www.webdav.org/specs/rfc4918.html#PROPPATCH-status
	Patch([]Proppatch) ([]Propstat, error)
}

// liveProps contains all supported, protected DAV: properties.
var liveProps = map[xml.Name]struct {
	// findFn implements the propfind function of this property. If nil,
	// it indicates a hidden property.
	findFn func(ali_driver.File) (string, error)
	// dir is true if the property applies to directories.
	dir bool
}{
	{Space: "DAV:", Local: "resourcetype"}: {
		findFn: findResourceType,
		dir:    true,
	},
	{Space: "DAV:", Local: "displayname"}: {
		findFn: findDisplayName,
		dir:    true,
	},
	{Space: "DAV:", Local: "getcontentlength"}: {
		findFn: findContentLength,
		dir:    false,
	},
	{Space: "DAV:", Local: "getlastmodified"}: {
		findFn: findLastModified,
		// http://webdav.org/specs/rfc4918.html#PROPERTY_getlastmodified
		// suggests that getlastmodified should only apply to GETable
		// resources, and this package does not support GET on directories.
		//
		// Nonetheless, some WebDAV clients expect child directories to be
		// sortable by getlastmodified date, so this value is true, not false.
		// See golang.org/issue/15334.
		dir: true,
	},
	{Space: "DAV:", Local: "creationdate"}: {
		findFn: nil,
		dir:    false,
	},
	{Space: "DAV:", Local: "getcontentlanguage"}: {
		findFn: nil,
		dir:    false,
	},
	{Space: "DAV:", Local: "getcontenttype"}: {
		findFn: findContentType,
		dir:    false,
	},
	{Space: "DAV:", Local: "getetag"}: {
		findFn: findETag,
		// findETag implements ETag as the concatenated hex values of a file's
		// modification time and size. This is not a reliable synchronization
		// mechanism for directories, so we do not advertise getetag for DAV
		// collections.
		dir: false,
	},
}

// TODO(nigeltao) merge props and allprop?

// Props returns the status of the properties named pnames for resource name.
//
// Each Propstat has a unique status and each property name will only be part
// of one Propstat element.
func props(fi ali_driver.File, pnames []xml.Name) ([]Propstat, error) {
	isDir := fi.IsDir()
	var deadProps map[xml.Name]Property

	pstatOK := Propstat{Status: http.StatusOK}
	pstatNotFound := Propstat{Status: http.StatusNotFound}
	for _, pn := range pnames {
		// If this file has dead properties, check if they contain pn.
		if dp, ok := deadProps[pn]; ok {
			pstatOK.Props = append(pstatOK.Props, dp)
			continue
		}
		// Otherwise, it must either be a live property or we don't know it.
		if prop := liveProps[pn]; prop.findFn != nil && (prop.dir || !isDir) {
			innerXML, err := prop.findFn(fi)
			if err != nil {
				return nil, err
			}
			pstatOK.Props = append(pstatOK.Props, Property{
				XMLName:  pn,
				InnerXML: []byte(innerXML),
			})
		} else {
			pstatNotFound.Props = append(pstatNotFound.Props, Property{
				XMLName: pn,
			})
		}
	}
	return makePropstats(pstatOK, pstatNotFound), nil
}

// Propnames returns the property names defined for resource name.
func propnames(fi ali_driver.File) ([]xml.Name, error) {

	isDir := fi.IsDir()

	var deadProps map[xml.Name]Property

	pnames := make([]xml.Name, 0, len(liveProps)+len(deadProps))
	for pn, prop := range liveProps {
		if prop.findFn != nil && (prop.dir || !isDir) {
			pnames = append(pnames, pn)
		}
	}
	for pn := range deadProps {
		pnames = append(pnames, pn)
	}
	return pnames, nil
}

// Allprop returns the properties defined for resource name and the properties
// named in include.
//
// Note that RFC 4918 defines 'allprop' to return the DAV: properties defined
// within the RFC plus dead properties. Other live properties should only be
// returned if they are named in 'include'.
//
// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPFIND
func allprop(fi ali_driver.File, include []xml.Name) ([]Propstat, error) {
	pnames, err := propnames(fi)
	if err != nil {
		return nil, err
	}
	// Add names from include if they are not already covered in pnames.
	nameset := make(map[xml.Name]bool)
	for _, pn := range pnames {
		nameset[pn] = true
	}
	for _, pn := range include {
		if !nameset[pn] {
			pnames = append(pnames, pn)
		}
	}
	return props(fi, pnames)
}

func escapeXML(s string) string {
	for i := 0; i < len(s); i++ {
		// As an optimization, if s contains only ASCII letters, digits or a
		// few special characters, the escaped value is s itself and we don't
		// need to allocate a buffer and convert between string and []byte.
		switch c := s[i]; {
		case c == ' ' || c == '_' ||
			('+' <= c && c <= '9') || // Digits as well as + , - . and /
			('A' <= c && c <= 'Z') ||
			('a' <= c && c <= 'z'):
			continue
		}
		// Otherwise, go through the full escaping process.
		var buf bytes.Buffer
		xml.EscapeText(&buf, []byte(s))
		return buf.String()
	}
	return s
}

func findResourceType(fi ali_driver.File) (string, error) {
	if fi.IsDir() {
		return `<D:collection xmlns:D="DAV:"/>`, nil
	}
	return "", nil
}

func slashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

func findDisplayName(fi ali_driver.File) (string, error) {
	if slashClean(fi.Name) == "/" {
		// Hide the real name of a possibly prefixed root directory.
		return "", nil
	}
	return escapeXML(fi.Name), nil
}

func findContentLength(fi ali_driver.File) (string, error) {
	return strconv.FormatInt(fi.Size, 10), nil
}

func findLastModified(fi ali_driver.File) (string, error) {
	return fi.UpdatedAt.UTC().Format(http.TimeFormat), nil
}

// ErrNotImplemented should be returned by optional interfaces if they
// want the original implementation to be used.
var ErrNotImplemented = errors.New("not implemented")

// ContentTyper is an optional interface for the os.FileInfo
// objects returned by the FileSystem.
//
// If this interface is defined then it will be used to read the
// content type from the object.
//
// If this interface is not defined the file will be opened and the
// content type will be guessed from the initial contents of the file.
type ContentTyper interface {
	// ContentType returns the content type for the file.
	//
	// If this returns error ErrNotImplemented then the error will
	// be ignored and the base implementation will be used
	// instead.
	ContentType(ctx context.Context) (string, error)
}

func findContentType(fi ali_driver.File) (string, error) {
	return mime.TypeByExtension(fi.ContentType), nil
}

// ETager is an optional interface for the os.FileInfo objects
// returned by the FileSystem.
//
// If this interface is defined then it will be used to read the ETag
// for the object.
//
// If this interface is not defined an ETag will be computed using the
// ModTime() and the Size() methods of the os.FileInfo object.
type ETager interface {
	// ETag returns an ETag for the file.  This should be of the
	// form "value" or W/"value"
	//
	// If this returns error ErrNotImplemented then the error will
	// be ignored and the base implementation will be used
	// instead.
	ETag(ctx context.Context) (string, error)
}

func findETag(fi ali_driver.File) (string, error) {
	// The Apache http 2.4 web server by default concatenates the
	// modification time and size of a file. We replicate the heuristic
	// with nanosecond granularity.

	return fmt.Sprintf(`"%x%x"`, fi.UpdatedAt.UnixNano(), fi.Size), nil
}
