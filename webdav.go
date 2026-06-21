package main

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type WebDAVClient struct {
	Backend *S3Client
}

func NewWebDAVClient() *WebDAVClient {
	return &WebDAVClient{
		Backend: NewS3Client(),
	}
}

func (h *WebDAVClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Logoutput(r.Method+" "+r.URL.Path, "debug")
	if (r.Method == "GET" || r.Method == "HEAD") && redirectCanonicalPath(w, r) {
		return
	}
	switch r.Method {
		case "GET":
			if r.URL.Path != "" && strings.HasSuffix(r.URL.Path, "/") {
				h.Get_html(w, r)
			} else {
				h.Get(w, r)
			}
		case "PROPFIND":
			h.Propfind(w, r)
		case "PUT":
			h.Put(w, r)
		case "DELETE":
			h.Delete(w, r)
		case "COPY":
			h.Copy(w, r)
		case "MOVE":
			h.Move(w, r)
		case "MKCOL":
			h.Mkcol(w, r)
		case "OPTIONS":
			h.Option(w, r)
		case "HEAD":
			h.Head(w, r)
		default:
			Logoutput("Method not allowed", "info")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WebDAVClient) Get(w http.ResponseWriter, r *http.Request) {
	key := objectKeyFromPath(r.URL.Path)
	result, actualKey, err := h.Backend.GetObjectWithFallback(key, r.URL.Path)
	if err != nil {
		Logoutput("Unable to Get object From Get Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer result.Body.Close()

	prefix, err := readContentPrefix(result.Body)
	if err != nil {
		Logoutput("Unable to read object prefix From Get Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setObjectResponseHeaders(w, actualKey, result.Metadata, result.ContentType, result.ContentDisposition, result.CacheControl, result.ContentEncoding, result.ContentLanguage, result.ETag, result.LastModified, result.Expires, result.ContentLength, prefix)

	_, err = io.Copy(w, io.MultiReader(bytes.NewReader(prefix), result.Body))
	if err != nil {
		Logoutput("Unable to io.copy object From Get Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *WebDAVClient) Get_html(w http.ResponseWriter, r *http.Request){
	keyPrefix := objectKeyFromPath(r.URL.Path)
	if keyPrefix != "" && !strings.HasSuffix(keyPrefix, "/") {
        keyPrefix += "/"
    }
	Logoutput("Get_html: "+keyPrefix, "debug")
	result, err := h.Backend.ListObjects(keyPrefix)
	if err != nil {
		Logoutput("Unable to List object From Get_html Requests: "+keyPrefix, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	displayPath := canonicalURLPath(r.URL.Path)
	html := `
	<html>
	<head>
		<title>Index of ` + displayPath + `</title>
		<style>
			th, td { text-align: left; padding: 0.5em; }
			th { border-bottom: 1px solid #eee; }
			body {
				background-color: black;
				color: white;
				font-family: sans-serif;
			}
			a {
				color: white;
			}
		</style>
	</head>
	<body>
		<h1>Index of ` + displayPath + `</h1>
		<table>
			<tr><th>Name</th><th>Last Modified</th><th>Size</th></tr>`

	parentpath := strings.Join(strings.Split(r.URL.Path, "/")[0:len(strings.Split(r.URL.Path, "/"))-2], "/")
	if keyPrefix == "/"{
		parentpath = ""
	}
	html += `<tr><td><a href="` + parentpath + "/" + `">../</a></td><td>-</td><td>-</td></tr>`

	for _, prefix := range result.CommonPrefixes {
		html += `<tr><td><a href="` + assetPathFromKey(*prefix.Prefix) + `">` + *prefix.Prefix + `</a></td><td>-</td><td>-</td></tr>`
	}
	for _, obj := range result.Contents {
		href := assetPathFromKey(*obj.Key)
		modified := obj.LastModified.String()
		modified = strings.Split(modified, ".")[0]
		size := formatByte(*obj.Size)
		html += `<tr><td><a href="` + href + `">` + *obj.Key + `</a></td><td>` + modified + `</td><td>` + size + `</td></tr>`
	}

	html += `</table></body></html>`
	Logoutput("HTML: "+html, "debug")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(html))
}

func (h *WebDAVClient) Propfind(w http.ResponseWriter, r *http.Request) {
	keyPrefix := objectKeyFromPath(r.URL.Path)
	if keyPrefix != "" && !strings.HasSuffix(keyPrefix, "/") {
        keyPrefix += "/"
    }
	Logoutput("Propfind: "+keyPrefix, "debug")
	result, err := h.Backend.ListObjects(keyPrefix)
	if err != nil {
		Logoutput("Unable to List object From Propfind Requests: "+keyPrefix, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml;charset=utf-8")
	w.Header().Set("Connection", "keep-alive")

	xmlResponse := `
	<?xml version="1.0" encoding="utf-8" ?>
	<d:multistatus xmlns:d="DAV:" xmlns:s="http://sabredav.org/ns">
		<d:response>
			<d:href>` + canonicalURLPath(r.URL.Path) + `</d:href>`

	for _, prefix := range result.CommonPrefixes {
		xmlResponse += `
		<d:response>
			<d:href>` + assetPathFromKey(*prefix.Prefix) + `</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>` + *prefix.Prefix + `</d:displayname>
					<d:resourcetype><d:collection/></d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>`
	}
	for _, obj := range result.Contents {
		modified := obj.LastModified.String()
		modified = strings.Split(modified, ".")[0]
		size := formatByte(*obj.Size)
		xmlResponse += `
		<d:response>
			<d:href>` + assetPathFromKey(*obj.Key) + `</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>` + path.Base(*obj.Key) + `</d:displayname>
					<d:getlastmodified>` + modified + `</d:getlastmodified>
					<d:getcontentlength>` + size + `</d:getcontentlength>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>`
	}

	xmlResponse += `</d:response>
	</d:multistatus>`
	xmlResponse = strings.Replace(xmlResponse, "\t", "", -1)
	xmlResponse = strings.Replace(xmlResponse, "\n", "", -1)
	xmlResponse = strings.Replace(xmlResponse, "  ", "", -1)
	Logoutput("XML: "+xmlResponse, "debug")
    w.WriteHeader(http.StatusMultiStatus)
    w.Write([]byte(xmlResponse))
}

func formatByte (size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " Bytes"
	}
	size = size / 1024
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " KB"
	}
	size = size / 1024
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " MB"
	}
	size = size / 1024
	return strconv.FormatInt(size, 10) + " GB"
}

func (h *WebDAVClient) Put(w http.ResponseWriter, r *http.Request) {
	key := objectKeyFromPath(r.URL.Path)
	Logoutput("Put: "+key, "debug")
	_, err := h.Backend.PutObject(key, r.Body, r.Header.Get("Content-Type"))
	if err != nil {
		Logoutput("Unable to Put object From Put Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WebDAVClient) Delete(w http.ResponseWriter, r *http.Request) {
	key := objectKeyFromPath(r.URL.Path)
	Logoutput("Delete: "+key, "debug")
	_, err := h.Backend.DeleteObject(key)
	if err != nil {
		Logoutput("Unable to Delete object From Delete Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WebDAVClient) Copy(w http.ResponseWriter, r *http.Request) {
	src := objectKeyFromPath(r.URL.Path)
	dest := r.Header.Get("Destination")
	if dest == "" {
		Logoutput("Destination header is required", "info")
		http.Error(w, "Destination header is required", http.StatusBadRequest)
		return
	}
	dest = objectKeyFromDestination(dest)
	Logoutput("Copy: "+src+" to "+dest, "debug")
	_, err := h.Backend.CopyObject(src, dest)
	if err != nil {
		Logoutput("Unable to Copy object From Copy Requests: "+src+" to "+dest, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WebDAVClient) Move(w http.ResponseWriter, r *http.Request) {
	src := objectKeyFromPath(r.URL.Path)
	dest := r.Header.Get("Destination")
	if dest == "" {
		Logoutput("Destination header is required", "info")
		http.Error(w, "Destination header is required", http.StatusBadRequest)
		return
	}
	dest = objectKeyFromDestination(dest)
	Logoutput("Move: "+src+" to "+dest, "debug")
	_, err := h.Backend.MoveObject(src, dest)
	if err != nil {
		Logoutput("Unable to Move object From Move Requests: "+src+" to "+dest, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WebDAVClient) Mkcol(w http.ResponseWriter, r *http.Request) {
	key := objectKeyFromPath(r.URL.Path)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	Logoutput("Mkcol: "+key, "debug")
	_, err := h.Backend.PutObject(key, strings.NewReader(""), "")
	if err != nil {
		Logoutput("Unable to Put object From Mkcol Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WebDAVClient) Option(w http.ResponseWriter, r *http.Request) {
	Logoutput("Option Requests", "debug")
	w.Header().Set("DAV", "1,2")
	w.Header().Set("Allow", "OPTIONS, GET, PUT, DELETE, COPY, MOVE, MKCOL, PROPFIND, HEAD")
	w.Header().Set("Content-Length", "0")
	w.Header().Set("MS-Author-Via", "DAV")
	w.WriteHeader(http.StatusOK)
}

func (h *WebDAVClient) Head(w http.ResponseWriter, r *http.Request) {
	key := objectKeyFromPath(r.URL.Path)
	Logoutput("Head: "+key, "debug")
	result, actualKey, err := h.Backend.GetObjectWithFallback(key, r.URL.Path)
	if err != nil {
		Logoutput("Unable to Get object From Head Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer result.Body.Close()

	prefix, err := readContentPrefix(result.Body)
	if err != nil {
		Logoutput("Unable to read object prefix From Head Requests: "+key, "info")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setObjectResponseHeaders(w, actualKey, result.Metadata, result.ContentType, result.ContentDisposition, result.CacheControl, result.ContentEncoding, result.ContentLanguage, result.ETag, result.LastModified, result.Expires, result.ContentLength, prefix)
	w.WriteHeader(http.StatusOK)
}

func redirectCanonicalPath(w http.ResponseWriter, r *http.Request) bool {
	canonicalPath := canonicalURLPath(r.URL.Path)
	if canonicalPath == r.URL.Path {
		return false
	}
	redirectURL := *r.URL
	redirectURL.Path = canonicalPath
	redirectURL.RawPath = ""
	http.Redirect(w, r, redirectURL.String(), http.StatusPermanentRedirect)
	return true
}

func canonicalURLPath(urlPath string) string {
	if urlPath == "" {
		return "/"
	}
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	for strings.Contains(urlPath, "//") {
		urlPath = strings.ReplaceAll(urlPath, "//", "/")
	}
	return urlPath
}

func objectKeyFromPath(urlPath string) string {
	return strings.TrimPrefix(canonicalURLPath(urlPath), "/")
}

func objectKeyFromDestination(destination string) string {
	parsedURL, err := url.Parse(destination)
	if err == nil && parsedURL.Path != "" {
		return objectKeyFromPath(parsedURL.Path)
	}
	return objectKeyFromPath(destination)
}

func assetPathFromKey(key string) string {
	return canonicalURLPath("/" + strings.TrimLeft(key, "/"))
}

func readContentPrefix(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	prefix := make([]byte, 512)
	n, err := io.ReadFull(body, prefix)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return prefix[:n], nil
}

func setObjectResponseHeaders(
	w http.ResponseWriter,
	key string,
	metadata map[string]*string,
	contentType *string,
	contentDisposition *string,
	cacheControl *string,
	contentEncoding *string,
	contentLanguage *string,
	etag *string,
	lastModified *time.Time,
	expires *string,
	contentLength *int64,
	prefix []byte,
) {
	for k, v := range metadata {
		if v == nil {
			continue
		}
		Logoutput(k+" : "+*v, "debug")
		w.Header().Set(k, *v)
	}

	resolvedContentType := objectContentType(key, stringValue(contentType), prefix)
	w.Header().Set("Content-Type", resolvedContentType)
	setOptionalHeader(w, "Content-Disposition", responseContentDisposition(stringValue(contentDisposition), resolvedContentType))
	setOptionalHeader(w, "Cache-Control", stringValue(cacheControl))
	setOptionalHeader(w, "Content-Encoding", stringValue(contentEncoding))
	setOptionalHeader(w, "Content-Language", stringValue(contentLanguage))
	setOptionalHeader(w, "ETag", stringValue(etag))
	if lastModified != nil {
		w.Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	}
	setOptionalHeader(w, "Expires", stringValue(expires))
	if contentLength != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*contentLength, 10))
	}
}

func responseContentDisposition(existingDisposition, contentType string) string {
	if !isImageContentType(contentType) {
		return existingDisposition
	}
	existingDisposition = strings.TrimSpace(existingDisposition)
	if existingDisposition == "" {
		return "inline"
	}
	if _, params, err := mime.ParseMediaType(existingDisposition); err == nil {
		if filename := params["filename"]; filename != "" {
			return `inline; filename="` + strings.ReplaceAll(filename, `"`, `\"`) + `"`
		}
	}
	parts := strings.SplitN(existingDisposition, ";", 2)
	if len(parts) == 2 {
		return "inline;" + parts[1]
	}
	return "inline"
}

func setOptionalHeader(w http.ResponseWriter, name string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	w.Header().Set(name, value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

