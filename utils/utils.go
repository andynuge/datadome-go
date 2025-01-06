package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datadome/module-go-package/models"
)

// GetMicroTime returns the current unix timestamp in microseconds
// This function needs to implement the time.UnixMicro function when Tyk will support the go version >= 1.18
func GetMicroTime() string {
	return strconv.FormatInt(time.Now().UnixNano()/1000, 10)
}

func GetIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	return ip, err
}

func GetHeaderList(r *http.Request) string {
	headerNames := make([]string, 0, len(r.Header))

	for headerName := range r.Header {
		headerNames = append(headerNames, headerName)
	}

	return strings.Join(headerNames, ",")
}

func GetURL(r *http.Request) string {
	if r.URL.RawQuery != "" {
		return r.URL.Path + "?" + r.URL.RawQuery
	} else {
		return r.URL.Path
	}
}

// cut implementation from the official package (see https://cs.opensource.google/go/go/+/master:src/strings/strings.go;l=1278)
func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// GetURI returns the URI without the query parameters nor the Fragments
// This function needs to implement the strings.Cut function when Tyk will support the go version >= 1.18
func GetURI(r *http.Request) string {
	var finalPath string

	pathWithoutQueryParams, _, _ := cut(r.URL.Path, "?")
	finalPath, _, _ = cut(pathWithoutQueryParams, "#")
	finalUri := r.URL.Host + finalPath

	return finalUri
}

type ApiFields string

const (
	Accept                 ApiFields = "Accept"
	AcceptCharset          ApiFields = "AcceptCharset"
	AcceptEncoding         ApiFields = "AcceptEncoding"
	AcceptLanguage         ApiFields = "AcceptLanguage"
	CacheControl           ApiFields = "CacheControl"
	ClientID               ApiFields = "ClientID"
	Connection             ApiFields = "Connection"
	ContentType            ApiFields = "ContentType"
	From                   ApiFields = "From"
	GraphQLOperationCount  ApiFields = "GraphQLOperationCount"
	GraphQLOperationName   ApiFields = "GraphQLOperationName"
	GraphQLOperationType   ApiFields = "GraphQLOperationType"
	HeadersList            ApiFields = "HeadersList"
	Host                   ApiFields = "Host"
	Origin                 ApiFields = "Origin"
	Pragma                 ApiFields = "Pragma"
	Referer                ApiFields = "Referer"
	Request                ApiFields = "Request"
	SecCHDeviceMemory      ApiFields = "SecCHDeviceMemory"
	SecCHUA                ApiFields = "SecCHUA"
	SecCHUAArch            ApiFields = "SecCHUAArch"
	SecCHUAFullVersionList ApiFields = "SecCHUAFullVersionList"
	SecCHUAMobile          ApiFields = "SecCHUAMobile"
	SecCHUAModel           ApiFields = "SecCHUAModel"
	SecCHUAPlatform        ApiFields = "SecCHUAPlatform"
	SecFetchUser           ApiFields = "SecFetchUser"
	SecFetchDest           ApiFields = "SecFetchDest"
	SecFetchMode           ApiFields = "SecFetchMode"
	SecFetchSite           ApiFields = "SecFetchSite"
	ServerHostname         ApiFields = "ServerHostname"
	ServerName             ApiFields = "ServerName"
	TrueClientIP           ApiFields = "TrueClientIP"
	UserAgent              ApiFields = "UserAgent"
	Via                    ApiFields = "Via"
	XForwardedForIP        ApiFields = "XForwardedForIP"
	XRealIP                ApiFields = "XRealIP"
	XRequestedWith         ApiFields = "XRequestedWith"
)

func getApiKeyLength(key ApiFields) int {
	switch key {
	case SecCHDeviceMemory, SecCHUAMobile, SecFetchUser:
		return 8
	case SecCHUAArch:
		return 16
	case SecCHUAPlatform, SecFetchDest, SecFetchMode:
		return 32
	case ContentType, SecFetchSite:
		return 64
	case CacheControl, ClientID, XRequestedWith, AcceptCharset, AcceptEncoding, Pragma, Connection, From, SecCHUA, SecCHUAModel, TrueClientIP, XRealIP, GraphQLOperationName:
		return 128
	case AcceptLanguage, SecCHUAFullVersionList, Via:
		return 256
	case HeadersList, Origin, ServerHostname, ServerName, Accept, Host:
		return 512
	case XForwardedForIP:
		return -512
	case UserAgent:
		return 768
	case Referer:
		return 1024
	case Request:
		return 2048
	}

	return -1
}

func GetApiKeyValue(key ApiFields, value string) string {
	if value == "" {
		return ""
	}

	limit := getApiKeyLength(key)
	if limit < -1 && len(value) > (-1*limit) {
		limit *= -1
		value = value[len(value)-limit:]
	} else if limit > -1 && len(value) > limit {
		value = value[:limit]
	}

	return value
}

func IsGraphQLRequest(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "application/json" && r.Method == "POST" && r.ContentLength > 0 && strings.Contains(r.URL.Path, "graphql")
}

func readBodyWithoutConsuming(r *http.Request, maximumBodySize int) ([]byte, error) {
	readSize := 1024
	regexp := regexp.MustCompile(`"query"\s*:\s*("(?:query|mutation|subscription)?\s*(?:[A-Za-z_][A-Za-z0-9_]*)?\s*[{(].*)`)
	limitedReader := &io.LimitedReader{
		R: r.Body,
		N: int64(maximumBodySize),
	}

	matchBuffer := make([]byte, 0, 2*readSize)
	buffer := make([]byte, readSize)
	var beginBody bytes.Buffer
	var matchedBytes []byte
	found := false

	for {
		n, err := limitedReader.Read(buffer)
		if n > 0 {
			beginBody.Write(buffer[:n])
			if !found {
				matchBuffer = append(matchBuffer, buffer[:n]...)
				if len(matchBuffer) >= 2*readSize {
					matchBuffer = matchBuffer[len(matchBuffer)-(2*readSize):]
				}
				matches := regexp.FindSubmatch(matchBuffer)
				if len(matches) > 1 {
					matchedBytes = matches[1]
					found = true
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	restBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(beginBody.Bytes()), bytes.NewReader(restBody)))

	return matchedBytes, nil
}

func ParseGraphQLQuery(body string) *models.GraphQLData {
	gqlData := &models.GraphQLData{
		Count: 0,
		Type:  models.Query,
		Name:  "",
	}
	regex := regexp.MustCompile(`(?m)(?P<operationType>query|mutation|subscription)\s*(?P<operationName>[A-Za-z_][A-Za-z0-9_]*)?\s*[({@]`)
	matches := regex.FindAllStringSubmatch(body, -1)
	matchLength := len(matches)

	if matchLength > 0 {
		gqlData.Count = matchLength
		firstMatch := matches[0]
		operationTypeIndex := regex.SubexpIndex("operationType")
		operationNameIndex := regex.SubexpIndex("operationName")

		if operationTypeIndex != -1 && operationTypeIndex < len(firstMatch) {
			gqlData.Type = models.GraphQLOperationType(firstMatch[operationTypeIndex])
		}
		if operationNameIndex != -1 && operationNameIndex < len(firstMatch) {
			gqlData.Name = firstMatch[operationNameIndex]
		}
	} else {
		shorthandSyntaxRegex := regexp.MustCompile(`"(?:query|mutation|subscription)?\s*(?:[A-Za-z_][A-Za-z0-9_]*)?\s*[{]`)
		shorthandMatches := shorthandSyntaxRegex.FindStringSubmatch(body)
		gqlData.Count = len(shorthandMatches)
	}

	return gqlData
}

func GetGraphQLData(r *http.Request, maximumBodySize int) *models.GraphQLData {
	defaultResult := models.GraphQLData{
		Count: 0,
		Type:  models.Query,
		Name:  "",
	}
	body, err := readBodyWithoutConsuming(r, maximumBodySize)
	if err != nil {
		log.Printf("error while reading request body: %v", err)
		return &defaultResult
	}
	if body == nil {
		log.Printf("query not found in the request body")
		return &defaultResult
	}
	bodyStr := string(body)

	return ParseGraphQLQuery(bodyStr)
}

func IsNullOrWhitespace(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func GetProtocol(r *http.Request) string {
	proto := "http"
	xForwardedProto := r.Header.Get("X-Forwarded-Proto")
	if strings.EqualFold(xForwardedProto, "http") || strings.EqualFold(xForwardedProto, "https") {
		proto = xForwardedProto
	} else if r.TLS != nil {
		proto = "https"
	}

	return proto
}

func sortQueryParams(u *url.URL) string {
	queryParams := u.Query()
	sortedQuery := url.Values{}
	for key, values := range queryParams {
		sort.Strings(values)
		sortedQuery[key] = values
	}
	u.RawQuery = queryParams.Encode()
	u.Fragment = ""

	return u.String()
}

func IsMatchingReferrer(r *http.Request) (bool, error) {
	fullURL := fmt.Sprintf("%s://%s%s", GetProtocol(r), r.Host, GetURL(r))
	parsedUrl, err := url.Parse(fullURL)
	if err != nil {
		fmt.Println(fmt.Errorf("fail to parse request URL: %w", err))
		return false, fmt.Errorf("fail to parse request URL: %w", err)
	}
	queryParams := parsedUrl.Query()
	queryParams.Del("dd_referrer")
	parsedUrl.RawQuery = queryParams.Encode()

	referer := r.Header.Get("Referer")
	if referer == "" {
		return false, nil
	}
	decodedReferer, err := url.QueryUnescape(referer)
	if err != nil {
		return false, fmt.Errorf("fail to decode referer header: %w", err)
	}
	parsedReferer, err := url.Parse(decodedReferer)
	if err != nil {
		return false, fmt.Errorf("fail to parse referer header: %w", err)
	}

	return sortQueryParams(parsedUrl) == sortQueryParams(parsedReferer), nil
}

func RestoreReferrer(r *http.Request) error {
	queryParams := r.URL.Query()
	if queryParams.Has("dd_referrer") {
		ddReferrer := queryParams.Get("dd_referrer")
		if ddReferrer == "" {
			r.Header.Del("Referer")
		} else {
			decodedReferrer, err := url.QueryUnescape(ddReferrer)
			if err != nil {
				return fmt.Errorf("fail to decoded dd_referrer query value: %w", err)
			}
			r.Header.Set("Referer", decodedReferrer)
		}
		queryParams.Del("dd_referrer")
		r.URL.RawQuery = queryParams.Encode()
	}
	return nil
}
