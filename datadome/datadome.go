package datadome

import (
	"fmt"
	"io"
	logger "log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/datadome/module-go-package/models"
	"github.com/datadome/module-go-package/utils"
	"github.com/google/go-querystring/query"
)

type DataDomeStruct models.DataDome

func NewClient(dd *DataDomeStruct) (*DataDomeStruct, error) {
	// Set the default configuration of Datadome
	config := dd

	if dd.DatadomeServerSideKey == "" {
		return nil, fmt.Errorf("the DatadomeServerSideKey must be defined")
	}
	if dd.DataDomeEndpoint == "" {
		config.DataDomeEndpoint = "api.datadome.co"
	}
	if dd.DataDomeTimeout == 0 {
		config.DataDomeTimeout = 150
	}
	if dd.UrlPatternExclusion == "" {
		config.UrlPatternExclusion = `(?i)\.(avi|flv|mka|mkv|mov|mp4|mpeg|mpg|mp3|flac|ogg|ogm|opus|wav|webm|webp|bmp|gif|ico|jpeg|jpg|png|svg|svgz|swf|eot|otf|ttf|woff|woff2|css|less|js|map|json)$`
	}
	if dd.ModuleName == "" {
		config.ModuleName = "Golang"
	}

	if dd.MaximumBodySize == 0 {
		dd.MaximumBodySize = 25 * 1024
	}

	if dd.ModuleVersion == "" {
		config.ModuleVersion = "1.3.0"
	}
	dd.Debug = config.Debug

	return config, nil
}

func h(w http.ResponseWriter, r *http.Request, next http.Handler, dd *DataDomeStruct) (bool, error) {
	// Logic before - reading request values, putting things into the
	// request context, performing bot mitigation

	sendNext := func(res bool, err error, response http.ResponseWriter) (bool, error) {
		if next != nil {
			next.ServeHTTP(response, r)
		} else {
			return res, err
		}
		return res, nil
	}

	dd, err := NewClient(dd)
	if err != nil {
		err = fmt.Errorf("error during creation of the DataDome client: %w", err)
		dd.log("%v\n", err)
		return sendNext(false, err, w)
	}

	uri := utils.GetURI(r)
	// Test exclusion regex
	if dd.UrlPatternExclusion != "" {
		re := regexp.MustCompile(dd.UrlPatternExclusion)
		if re.FindString(uri) != "" {
			dd.log("UrlPatternExclusion matches requested URI, skipping.\n")
			return sendNext(false, nil, w)
		}
	}

	// Test inclusion regex
	if dd.UrlPatternInclusion != "" {
		re := regexp.MustCompile(dd.UrlPatternInclusion)
		if re.FindString(uri) == "" {
			dd.log("UrlPatternInclusion does not match requested URI, skipping.\n")
			return sendNext(false, nil, w)
		}
	}

	queryStr, err := dd.buildRequest(r, dd.ModuleName)
	if err != nil {
		dd.log("error when building DataDome request: %v", err)
		return sendNext(false, err, w)
	}

	err, resp, isBlocked := dd.datadomeCall(queryStr, r, w)
	if err != nil {
		dd.log("error when performing call to DataDome API: %v", err)
		return sendNext(isBlocked, err, resp)
	}
	return sendNext(isBlocked, nil, resp)
	// Logic after - useful for logging, metrics, etc.
	//
	// It's important that we don't use the ResponseWriter after we've called the
	// next handler: we may cause conflicts when trying to write the response
}

func (dd *DataDomeStruct) DatadomeHandler(next http.Handler) http.Handler {
	// Wrap our anonymous function, and cast it to a http.HandlerFunc
	// Because our function signature matches ServeHTTP(w, r), this allows
	// our function (type) to implicitly satisify the http.Handler interface.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := h(w, r, next, dd)
		if err != nil {
			panic(err)
		}
	})
}

func (dd *DataDomeStruct) DatadomeProtect(rw http.ResponseWriter, r *http.Request) (isBlocked bool, err error) {
	// Wrap our anonymous function, and cast it to a http.HandlerFunc
	// Because our function signature matches ServeHTTP(w, r), this allows
	// our function (type) to implicitly satisify the http.Handler interface.
	return h(rw, r, nil, dd)
}

func (dd *DataDomeStruct) buildRequest(r *http.Request, moduleName string) (string, error) {
	// Build DataDome request with the original request
	// Todo: Key, IP, then the rest should be sorted
	contentLength := "0"
	if r.Header.Get("content-length") != "" {
		contentLength = r.Header.Get("content-length")
	}

	authorizationLen := "0"
	if r.Header.Get("Authorization") != "" {
		authorizationLen = strconv.Itoa(len(r.Header.Get("Authorization")))
	}

	proto := utils.GetProtocol(r)

	port := r.URL.Port()
	if port == "" {
		if proto == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	cookiesLen := "0"
	if r.Header.Get("Cookie") != "" {
		cookiesLen = strconv.Itoa(len(r.Header.Get("Cookie")))
	}

	ip, err := utils.GetIP(r)
	if err != nil {
		return "", fmt.Errorf("fail to parse request's IP: %w", err)
	}

	if dd.EnableReferrerRestoration {
		isMatching, err := utils.IsMatchingReferrer(r)
		if err != nil {
			dd.log("fail to check if the referrer matches: %v", err)
		} else if isMatching {
			err = utils.RestoreReferrer(r)
			if err != nil {
				dd.log("fail to restore the referrer: %v", err)
			}
		}
	}

	ddRequestParams := models.DataDomeRequest{
		Key:                    dd.DatadomeServerSideKey,
		IP:                     ip,
		Accept:                 utils.GetApiKeyValue(utils.Accept, r.Header.Get("accept")),
		AcceptCharset:          utils.GetApiKeyValue(utils.AcceptCharset, r.Header.Get("accept-charset")),
		AcceptEncoding:         utils.GetApiKeyValue(utils.AcceptEncoding, r.Header.Get("accept-encoding")),
		AcceptLanguage:         utils.GetApiKeyValue(utils.AcceptLanguage, r.Header.Get("accept-language")),
		APIConnectionState:     "new",
		AuthorizationLen:       authorizationLen,
		CacheControl:           utils.GetApiKeyValue(utils.CacheControl, r.Header.Get("cache-control")),
		ClientID:               utils.GetApiKeyValue(utils.ClientID, getClientId(r)),
		Connection:             utils.GetApiKeyValue(utils.Connection, r.Header.Get("connection")),
		ContentType:            utils.GetApiKeyValue(utils.ContentType, r.Header.Get("Content-Type")),
		CookiesLen:             cookiesLen,
		From:                   utils.GetApiKeyValue(utils.From, r.Header.Get("From")),
		HeadersList:            utils.GetApiKeyValue(utils.HeadersList, utils.GetHeaderList(r)),
		Host:                   utils.GetApiKeyValue(utils.Host, r.Host),
		Method:                 r.Method,
		ModuleVersion:          dd.ModuleVersion,
		Origin:                 utils.GetApiKeyValue(utils.Origin, r.Header.Get("origin")),
		Port:                   port,
		PostParamLen:           contentLength,
		Pragma:                 utils.GetApiKeyValue(utils.Pragma, r.Header.Get("pragma")),
		Protocol:               proto,
		Referer:                utils.GetApiKeyValue(utils.Referer, r.Header.Get("referer")),
		Request:                utils.GetApiKeyValue(utils.Request, utils.GetURL(r)),
		RequestModuleName:      moduleName,
		SecChDeviceMemory:      utils.GetApiKeyValue(utils.SecCHDeviceMemory, r.Header.Get("Sec-CH-Device-Memory")),
		SecChUA:                utils.GetApiKeyValue(utils.SecCHUA, r.Header.Get("Sec-CH-UA")),
		SecChUAArch:            utils.GetApiKeyValue(utils.SecCHUAArch, r.Header.Get("Sec-CH-UA-Arch")),
		SecChUAFullVersionList: utils.GetApiKeyValue(utils.SecCHUAFullVersionList, r.Header.Get("Sec-CH-UA-Full-Version-List")),
		SecChUAMobile:          utils.GetApiKeyValue(utils.SecCHUAMobile, r.Header.Get("Sec-CH-UA-Mobile")),
		SecChUAModel:           utils.GetApiKeyValue(utils.SecCHUAModel, r.Header.Get("Sec-CH-UA-Model")),
		SecChUAPlatform:        utils.GetApiKeyValue(utils.SecCHUAPlatform, r.Header.Get("Sec-CH-UA-Platform")),
		SecFetchDest:           utils.GetApiKeyValue(utils.SecFetchDest, r.Header.Get("Sec-Fetch-Dest")),
		SecFetchMode:           utils.GetApiKeyValue(utils.SecFetchMode, r.Header.Get("Sec-Fetch-Mode")),
		SecFetchSite:           utils.GetApiKeyValue(utils.SecFetchSite, r.Header.Get("Sec-Fetch-Site")),
		SecFetchUser:           utils.GetApiKeyValue(utils.SecFetchUser, r.Header.Get("Sec-Fetch-User")),
		ServerHostName:         utils.GetApiKeyValue(utils.ServerHostname, r.Host),
		ServerName:             utils.GetApiKeyValue(utils.ServerName, r.Host),
		TimeRequest:            utils.GetMicroTime(),
		TrueClientIP:           utils.GetApiKeyValue(utils.TrueClientIP, r.Header.Get("True-Client-IP")),
		UserAgent:              utils.GetApiKeyValue(utils.UserAgent, r.Header.Get("user-agent")),
		Via:                    utils.GetApiKeyValue(utils.Via, r.Header.Get("Via")),
		XForwardedForIP:        utils.GetApiKeyValue(utils.XForwardedForIP, r.Header.Get("x-forwarded-for")),
		XRealIP:                utils.GetApiKeyValue(utils.XRealIP, r.Header.Get("X-Real-IP")),
		XRequestedWith:         utils.GetApiKeyValue(utils.XRequestedWith, r.Header.Get("X-Requested-With")),
	}

	if dd.EnableGraphQLSupport && utils.IsGraphQLRequest(r) {
		gqlData := utils.GetGraphQLData(r, dd.MaximumBodySize)
		if gqlData.Count != 0 {
			operationName := utils.GetApiKeyValue(utils.GraphQLOperationName, gqlData.Name)
			ddRequestParams.GraphQLOperationName = &operationName
			ddRequestParams.GraphQLOperationType = gqlData.Type
			ddRequestParams.GraphQLOperationCount = strconv.Itoa(gqlData.Count)
		}
	}

	queryStr, err := query.Values(&ddRequestParams)
	if err != nil {
		return "", fmt.Errorf("fail to set query values: %w", err)
	}

	return queryStr.Encode(), nil
}

func (dd *DataDomeStruct) log(format string, v ...interface{}) {
	if dd.Debug {
		logger.Printf(format, v...)
	}
}

// datadomeCall
func (dd *DataDomeStruct) datadomeCall(jsonStr string, origReq *http.Request, origResp http.ResponseWriter) (err error, rw http.ResponseWriter, isBlocked bool) {
	// Create and send a request to DataDome API Server and handle the X-Datadomeresponse code
	client := &http.Client{
		Timeout: time.Millisecond * time.Duration(dd.DataDomeTimeout),
	}
	body := strings.NewReader(jsonStr)
	datadomeEndpoint := dd.DataDomeEndpoint
	if !strings.HasPrefix(datadomeEndpoint, "http") && !strings.HasPrefix(datadomeEndpoint, "/") {
		datadomeEndpoint = fmt.Sprintf("https://%s/validate-request", datadomeEndpoint)
	}
	req, err := http.NewRequestWithContext(origReq.Context(), "POST", datadomeEndpoint, body)
	if err != nil {
		return fmt.Errorf("error when instancing new DataDome request %w", err), nil, true
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "DataDome")

	if origReq.Header.Get("x-datadome-clientid") != "" {
		req.Header.Set("X-DataDome-X-Set-Cookie", "true")
	}

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error when performing DataDome request: %w", err), nil, false
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error when reading DataDome response %w", err), nil, false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			dd.log("error when closing the Body: %v\n", err)
		}
	}(response.Body)

	ddStatus := response.Header.Get("X-Datadomeresponse")
	ddRespStatus := strconv.Itoa(response.StatusCode)

	if ddStatus == "" || (ddRespStatus != ddStatus) {
		dd.log("fail to get status code and response headers from DataDome API response. reason: %v\n", string(b))
		return fmt.Errorf("fails to get status code and response headers from DataDome API response. Bypass DataDome. Full DataDome response: %v", response), nil, false
	}

	// Handler DataDome status code
	if ddStatus == "400" {
		return nil, origResp, false
	} else if ddStatus == "301" || ddStatus == "302" {
		fmt.Println("Not implemented...")
		return nil, origResp, false

	} else if ddStatus == "401" || ddStatus == "403" {
		origResp = addDataDomeHeaders(response, origResp)
		origResp.WriteHeader(http.StatusForbidden)
		_, err := origResp.Write(b)
		if err != nil {
			return err, nil, false
		}
		return nil, origResp, true

	} else if ddStatus == "200" {
		addDataDomeRequestHeaders(response, origReq)
		origResp = addDataDomeHeaders(response, origResp)
		return nil, origResp, false

	} else {
		dd.log("%s response from DataDome API - Unexpected error. If the error remains, please contact us at support@datadome.co. Full response: %v", ddStatus, response.Header)
		return nil, origResp, false
	}
}

func addDataDomeRequestHeaders(ddResp *http.Response, origReq *http.Request) {
	// From the DataDome response, we will read requests headers and add them to
	// the original request
	datadomeHeadersStr := ddResp.Header.Get("x-datadome-request-headers")
	if datadomeHeadersStr != "" {
		datadomeHeaders := strings.Fields(datadomeHeadersStr)
		for _, datadomeHeaderName := range datadomeHeaders {
			datadomeHeaderValue := ddResp.Header.Get(datadomeHeaderName)
			if datadomeHeaderValue != "" {
				origReq.Header.Add(datadomeHeaderName, datadomeHeaderValue)
			}
		}
	}
}

func addDataDomeHeaders(ddResp *http.Response, origResp http.ResponseWriter) http.ResponseWriter {
	// From the DataDome response, we will read datadome headers and add them to
	// the original response (X-DataDome and Set-Cookie)
	datadomeHeadersStr := ddResp.Header.Get("x-datadome-headers")
	if datadomeHeadersStr != "" {
		datadomeHeaders := strings.Fields(datadomeHeadersStr)
		for _, datadomeHeaderName := range datadomeHeaders {
			datadomeHeaderValue := ddResp.Header.Get(datadomeHeaderName)
			if datadomeHeaderValue != "" {
				origResp.Header().Set(datadomeHeaderName, datadomeHeaderValue)
			}
		}
	}
	return origResp
}

func getClientId(r *http.Request) string {
	clientIDHeaders := r.Header.Get("x-datadome-clientid")
	if len(clientIDHeaders) > 0 {
		return clientIDHeaders
	}

	cookie, err := r.Cookie("datadome")
	if err == nil {
		return cookie.Value
	}

	return ""
}
