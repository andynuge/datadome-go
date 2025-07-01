package modulego

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NewClient instantiate a new DataDome [Client] to perform calls to Protection API.
// The fields may be customized through [Option] functions.
// It returns an error in case of [incorrect / invalid] inputs in the options.
func NewClient(serverSideKey string, options ...Option) (*Client, error) {
	c := &Client{
		EnableGraphQLSupport:      DefaultEnableGraphQLSupportValue,
		EnableReferrerRestoration: DefaultEnableReferrerRestorationValue,
		Endpoint:                  DefaultEndpointValue,
		Logger:                    NewDefaultLogger(),
		MaximumBodySize:           DefaultMaximumBodySizeValue,
		ModuleName:                DefaultModuleNameValue,
		ModuleVersion:             DefaultModuleVersionValue,
		ServerSideKey:             serverSideKey,
		Timeout:                   DefaultTimeoutValue,
		UrlPatternInclusion:       DefaultUrlPatternInclusionValue,
		UrlPatternExclusion:       DefaultUrlPatternExclusionValue,
	}

	// apply functional options
	for _, opt := range options {
		opt(c)
	}

	// error management
	if c.ServerSideKey == "" {
		return nil, fmt.Errorf("ServerSideKey must be defined")
	}
	if c.Timeout <= 0 {
		return nil, fmt.Errorf("Timeout must be a positive integer")
	}
	if c.MaximumBodySize <= 0 {
		return nil, fmt.Errorf("MaximumBodySize must be a positive integer")
	}

	// set not exported values
	c.httpClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Millisecond * time.Duration(c.Timeout),
	}
	if c.UrlPatternExclusion != "" {
		r, err := regexp.Compile(c.UrlPatternExclusion)
		if err != nil {
			return nil, fmt.Errorf("UrlPatternExclusion must be a valid RegExp: %w", err)
		}
		c.urlPatternExclusion = r
	}
	if c.UrlPatternInclusion != "" {
		r, err := regexp.Compile(c.UrlPatternInclusion)
		if err != nil {
			return nil, fmt.Errorf("UrlPatternInclusion must be a valid RegExp: %w", err)
		}
		c.urlPatternInclusion = r
	}
	c.endpoint = c.Endpoint
	if !strings.HasPrefix(c.Endpoint, "http") && !strings.HasPrefix(c.Endpoint, "/") {
		c.endpoint = fmt.Sprintf("https://%s/validate-request", c.Endpoint)
	}

	return c, nil
}

// handler is used to validate incoming requests
// This function will:
// 1. Verifies the request URL does not match the UrlPatternExclusion
// 2. Verifies the request URL match the UrlPatternInclusion (if set)
// 3. Builds the request payload for the Protection API
// 4. Performs the call to the Protection API and interpret the response
func (c *Client) handler(w http.ResponseWriter, r *http.Request, next http.Handler) (bool, error) {
	sendNext := func(res bool, err error, response http.ResponseWriter) (bool, error) {
		if next != nil {
			next.ServeHTTP(response, r)
		} else {
			return res, err
		}
		return res, nil
	}

	uri := getURI(r)
	// Test exclusion regex
	if c.urlPatternExclusion != nil && c.urlPatternExclusion.MatchString(uri) {
		c.Logger.Info("UrlPatternExclusion matches requested URI, skipping.")
		return false, nil
	}

	// Test inclusion regex
	if c.urlPatternInclusion != nil && !c.urlPatternInclusion.MatchString(uri) {
		c.Logger.Info("UrlPatternInclusion does not match requested URI, skipping.")
		return false, nil
	}

	queryStr, err := c.buildRequest(r)
	if err != nil {
		c.Logger.Error("error when building request payload: %v", err)
		return sendNext(false, err, w)
	}

	err, resp, isBlocked := c.datadomeCall(queryStr, r, w)
	if err != nil {
		c.Logger.Error("error when performing call to Protection API: %v", err)
		return sendNext(isBlocked, err, resp)
	}
	return sendNext(isBlocked, nil, resp)
}

// DatadomeHandler implements the [http.Handler] interface
func (c *Client) DatadomeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := c.handler(w, r, next)
		if err != nil {
			panic(err)
		}
	})
}

// DatadomeProtect validates the incoming request
func (c *Client) DatadomeProtect(rw http.ResponseWriter, r *http.Request) (isBlocked bool, err error) {
	return c.handler(rw, r, nil)
}

// buildRequest extracts information from the request and build the payload to be sent to the Protection API.
// An error may be returned if the IP cannot be retrieved or if it fails to URL-encode the payload.
func (c *Client) buildRequest(r *http.Request) (string, error) {
	// Build DataDome request with the original request
	contentLength := "0"
	if r.Header.Get("content-length") != "" {
		contentLength = r.Header.Get("content-length")
	}

	authorizationLen := "0"
	if r.Header.Get("authorization") != "" {
		authorizationLen = strconv.Itoa(len(r.Header.Get("authorization")))
	}

	proto := getProtocol(r)

	port := r.URL.Port()
	if port == "" {
		if proto == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	host := r.Host
	if c.UseXForwardedHost {
		host = getHost(r)
	}

	cookiesLen := "0"
	if r.Header.Get("Cookie") != "" {
		cookiesLen = strconv.Itoa(len(r.Header.Get("cookie")))
	}

	cookiesList := getCookieList(r)

	ip, err := getIP(r)
	if err != nil {
		return "", fmt.Errorf("fail to parse request's IP: %w", err)
	}

	if c.EnableReferrerRestoration {
		isMatching, err := isMatchingReferrer(r)
		if err != nil {
			c.Logger.Warn("fail to check if the referrer matches: %v", err)
		} else if isMatching {
			err = restoreReferrer(r)
			if err != nil {
				c.Logger.Warn("fail to restore the referrer: %v", err)
			}
		}
	}

	ddRequestParams := ProtectionAPIRequestPayload{
		Key:                    c.ServerSideKey,
		IP:                     ip,
		Accept:                 truncateValue(Accept, r.Header.Get("accept")),
		AcceptCharset:          truncateValue(AcceptCharset, r.Header.Get("accept-charset")),
		AcceptEncoding:         truncateValue(AcceptEncoding, r.Header.Get("accept-encoding")),
		AcceptLanguage:         truncateValue(AcceptLanguage, r.Header.Get("accept-language")),
		APIConnectionState:     "new",
		AuthorizationLen:       authorizationLen,
		CacheControl:           truncateValue(CacheControl, r.Header.Get("cache-control")),
		ClientID:               truncateValue(ClientID, getClientId(r)),
		Connection:             truncateValue(Connection, r.Header.Get("connection")),
		ContentType:            truncateValue(ContentType, r.Header.Get("content-type")),
		CookiesLen:             cookiesLen,
		CookiesList:            cookiesList,
		From:                   truncateValue(From, r.Header.Get("from")),
		HeadersList:            truncateValue(HeadersList, getHeaderList(r)),
		Host:                   truncateValue(Host, host),
		Method:                 r.Method,
		ModuleVersion:          c.ModuleVersion,
		Origin:                 truncateValue(Origin, r.Header.Get("origin")),
		Port:                   port,
		PostParamLen:           contentLength,
		Pragma:                 truncateValue(Pragma, r.Header.Get("pragma")),
		Protocol:               proto,
		Referer:                truncateValue(Referer, r.Header.Get("referer")),
		Request:                truncateValue(Request, getURL(r)),
		RequestModuleName:      c.ModuleName,
		SecChDeviceMemory:      truncateValue(SecCHDeviceMemory, r.Header.Get("sec-ch-device-memory")),
		SecChUA:                truncateValue(SecCHUA, r.Header.Get("sec-ch-ua")),
		SecChUAArch:            truncateValue(SecCHUAArch, r.Header.Get("sec-ch-ua-arch")),
		SecChUAFullVersionList: truncateValue(SecCHUAFullVersionList, r.Header.Get("sec-ch-ua-full-version-list")),
		SecChUAMobile:          truncateValue(SecCHUAMobile, r.Header.Get("sec-ch-ua-mobile")),
		SecChUAModel:           truncateValue(SecCHUAModel, r.Header.Get("sec-ch-ua-model")),
		SecChUAPlatform:        truncateValue(SecCHUAPlatform, r.Header.Get("sec-ch-ua-platform")),
		SecFetchDest:           truncateValue(SecFetchDest, r.Header.Get("sec-fetch-dest")),
		SecFetchMode:           truncateValue(SecFetchMode, r.Header.Get("sec-fetch-mode")),
		SecFetchSite:           truncateValue(SecFetchSite, r.Header.Get("sec-fetch-site")),
		SecFetchUser:           truncateValue(SecFetchUser, r.Header.Get("sec-fetch-user")),
		ServerHostName:         truncateValue(ServerHostname, host),
		ServerName:             truncateValue(ServerName, host),
		TimeRequest:            getMicroTime(),
		TrueClientIP:           truncateValue(TrueClientIP, r.Header.Get("true-client-ip")),
		UserAgent:              truncateValue(UserAgent, r.Header.Get("user-agent")),
		Via:                    truncateValue(Via, r.Header.Get("via")),
		XForwardedForIP:        truncateValue(XForwardedForIP, r.Header.Get("x-forwarded-for")),
		XRealIP:                truncateValue(XRealIP, r.Header.Get("x-real-ip")),
		XRequestedWith:         truncateValue(XRequestedWith, r.Header.Get("x-requested-with")),
	}

	if c.EnableGraphQLSupport && isGraphQLRequest(r) {
		gqlData, err := getGraphQLData(r, c.MaximumBodySize)
		if err != nil {
			c.Logger.Warn("fail to retrieve GraphQL data: %v", err)
		}
		if gqlData != nil && gqlData.Count != 0 {
			operationName := truncateValue(GraphQLOperationName, gqlData.Name)
			ddRequestParams.GraphQLOperationName = &operationName
			ddRequestParams.GraphQLOperationType = gqlData.Type
			ddRequestParams.GraphQLOperationCount = strconv.Itoa(gqlData.Count)
		}
	}

	queryStr := buildQuery(&ddRequestParams)

	return queryStr.Encode(), nil
}

// datadomeCall performs a request to the Protection API
func (c *Client) datadomeCall(jsonStr string, origReq *http.Request, origResp http.ResponseWriter) (err error, rw http.ResponseWriter, isBlocked bool) {
	body := strings.NewReader(jsonStr)
	req, err := http.NewRequestWithContext(origReq.Context(), "POST", c.endpoint, body)
	if err != nil {
		return fmt.Errorf("error when instancing new DataDome request %w", err), nil, false
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("user-agent", "DataDome")

	if origReq.Header.Get("x-datadome-clientid") != "" {
		req.Header.Set("x-datadome-x-set-cookie", "true")
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error when performing DataDome request: %w", err), nil, false
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error when reading DataDome response %w", err), nil, false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			c.Logger.Warn("error when closing the Body: %v", err)
		}
	}(response.Body)

	ddStatus := response.Header.Get("x-datadomeresponse")
	ddRespStatus := strconv.Itoa(response.StatusCode)

	if ddStatus == "" || (ddRespStatus != ddStatus) {
		c.Logger.Debug("fail to get status code and response headers from Protection API response. reason: %s", string(responseBody))
		return fmt.Errorf("fails to get status code and response headers from Protection API response. Bypass DataDome. Full DataDome response: %v", response), nil, false
	}

	// Handler DataDome status code
	if ddStatus == "400" {
		return nil, origResp, false
	} else if ddStatus == "301" || ddStatus == "302" || ddStatus == "401" || ddStatus == "403" {
		origResp = addDataDomeHeaders(response, origResp)
		origResp.WriteHeader(response.StatusCode)
		_, err = origResp.Write(responseBody)
		if err != nil {
			return err, nil, false
		}
		return nil, origResp, true

	} else if ddStatus == "200" {
		addDataDomeRequestHeaders(response, origReq)
		origResp = addDataDomeHeaders(response, origResp)
		return nil, origResp, false

	} else {
		return fmt.Errorf("%s response from Protection API - Unexpected error. If the error remains, please contact us at support@datadome.co. Full response: %v", ddStatus, response.Header), origResp, false
	}
}

// addDataDomeRequestHeaders add the headers listed in the `X-datadome-request-headers`
// header of the Protection API response to the original request.
func addDataDomeRequestHeaders(ddResp *http.Response, origReq *http.Request) {
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

// addDataDomeHeaders add the headers listed in the `x-datadome-headers` header
// of the Protection API response to the original response.
func addDataDomeHeaders(ddResp *http.Response, origResp http.ResponseWriter) http.ResponseWriter {
	datadomeHeadersStr := ddResp.Header.Get("x-datadome-headers")
	if datadomeHeadersStr != "" {
		datadomeHeaders := strings.Fields(datadomeHeadersStr)
		for _, datadomeHeaderName := range datadomeHeaders {
			datadomeHeaderValue := ddResp.Header.Get(datadomeHeaderName)
			if datadomeHeaderValue != "" {
				if strings.EqualFold(datadomeHeaderName, "set-cookie") {
					origResp.Header().Add(datadomeHeaderName, datadomeHeaderValue)
				} else {
					origResp.Header().Set(datadomeHeaderName, datadomeHeaderValue)
				}
			}
		}
	}
	return origResp
}

// getClientId retrieves the ClientID from the incoming request.
// It uses the value of the `X-DataDome-ClientID` if the session by header feature is used.
// It reads the `DataDome` cookie value otherwise.
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

func buildQuery(p *ProtectionAPIRequestPayload) url.Values {
	values := url.Values{}

	// Required fields
	values.Set("Key", p.Key)
	values.Set("RequestModuleName", p.RequestModuleName)
	values.Set("IP", p.IP)
	values.Set("Request", p.Request)

	// Optional fields
	if p.ModuleVersion != "" {
		values.Set("ModuleVersion", p.ModuleVersion)
	}
	if p.ServerName != "" {
		values.Set("ServerName", p.ServerName)
	}
	if p.APIConnectionState != "" {
		values.Set("APIConnectionState", p.APIConnectionState)
	}
	if p.Port != "" {
		values.Set("Port", p.Port)
	}
	if p.TimeRequest != "" {
		values.Set("TimeRequest", p.TimeRequest)
	}
	if p.Protocol != "" {
		values.Set("Protocol", p.Protocol)
	}
	if p.Method != "" {
		values.Set("Method", p.Method)
	}
	if p.ServerHostName != "" {
		values.Set("ServerHostname", p.ServerHostName)
	}
	if p.HeadersList != "" {
		values.Set("HeadersList", p.HeadersList)
	}
	if p.Host != "" {
		values.Set("Host", p.Host)
	}
	if p.UserAgent != "" {
		values.Set("UserAgent", p.UserAgent)
	}
	if p.Referer != "" {
		values.Set("Referer", p.Referer)
	}
	if p.Accept != "" {
		values.Set("Accept", p.Accept)
	}
	if p.AcceptEncoding != "" {
		values.Set("AcceptEncoding", p.AcceptEncoding)
	}
	if p.AcceptLanguage != "" {
		values.Set("AcceptLanguage", p.AcceptLanguage)
	}
	if p.AcceptCharset != "" {
		values.Set("AcceptCharset", p.AcceptCharset)
	}
	if p.Origin != "" {
		values.Set("Origin", p.Origin)
	}
	if p.XForwardedForIP != "" {
		values.Set("XForwardedForIP", p.XForwardedForIP)
	}
	if p.XRequestedWith != "" {
		values.Set("X-Requested-With", p.XRequestedWith)
	}
	if p.Connection != "" {
		values.Set("Connection", p.Connection)
	}
	if p.Pragma != "" {
		values.Set("Pragma", p.Pragma)
	}
	if p.CacheControl != "" {
		values.Set("CacheControl", p.CacheControl)
	}
	if p.CookiesLen != "" {
		values.Set("CookiesLen", p.CookiesLen)
	}
	if p.CookiesList != "" {
		values.Set("CookiesList", p.CookiesList)
	}
	if p.AuthorizationLen != "" {
		values.Set("AuthorizationLen", p.AuthorizationLen)
	}
	if p.PostParamLen != "" {
		values.Set("PostParamLen", p.PostParamLen)
	}
	if p.XRealIP != "" {
		values.Set("X-Real-IP", p.XRealIP)
	}
	if p.ClientID != "" {
		values.Set("ClientID", p.ClientID)
	}
	if p.SecChDeviceMemory != "" {
		values.Set("SecCHDeviceMemory", p.SecChDeviceMemory)
	}
	if p.SecChUA != "" {
		values.Set("SecCHUA", p.SecChUA)
	}
	if p.SecChUAArch != "" {
		values.Set("SecCHUAArch", p.SecChUAArch)
	}
	if p.SecChUAFullVersionList != "" {
		values.Set("SecCHUAFullVersionList", p.SecChUAFullVersionList)
	}
	if p.SecChUAMobile != "" {
		values.Set("SecCHUAMobile", p.SecChUAMobile)
	}
	if p.SecChUAModel != "" {
		values.Set("SecCHUAModel", p.SecChUAModel)
	}
	if p.SecChUAPlatform != "" {
		values.Set("SecCHUAPlatform", p.SecChUAPlatform)
	}
	if p.SecFetchDest != "" {
		values.Set("SecFetchDest", p.SecFetchDest)
	}
	if p.SecFetchMode != "" {
		values.Set("SecFetchMode", p.SecFetchMode)
	}
	if p.SecFetchSite != "" {
		values.Set("SecFetchSite", p.SecFetchSite)
	}
	if p.SecFetchUser != "" {
		values.Set("SecFetchUser", p.SecFetchUser)
	}
	if p.Via != "" {
		values.Set("Via", p.Via)
	}
	if p.From != "" {
		values.Set("From", p.From)
	}
	if p.ContentType != "" {
		values.Set("ContentType", p.ContentType)
	}
	if p.TrueClientIP != "" {
		values.Set("TrueClientIP", p.TrueClientIP)
	}
	if p.GraphQLOperationCount != "" {
		values.Set("GraphQLOperationCount", p.GraphQLOperationCount)
	}
	if p.GraphQLOperationName != nil {
		values.Set("GraphQLOperationName", *p.GraphQLOperationName)
	}
	if p.GraphQLOperationType != "" {
		values.Set("GraphQLOperationType", string(p.GraphQLOperationType))
	}

	return values
}
