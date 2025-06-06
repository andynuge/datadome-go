package modulego

import (
	"net/http"
	"regexp"
)

const (
	DefaultEnableGraphQLSupportValue      = false
	DefaultEnableReferrerRestorationValue = false
	DefaultEndpointValue                  = "api.datadome.co"
	DefaultMaximumBodySizeValue           = 25 * 1024
	DefaultModuleNameValue                = "Golang"
	DefaultModuleVersionValue             = "2.2.0"
	DefaultTimeoutValue                   = 150
	DefaultUrlPatternInclusionValue       = ""
	DefaultUrlPatternExclusionValue       = `(?i)\.(avi|avif|bmp|css|eot|flac|flv|gif|gz|ico|jpeg|jpg|js|json|less|map|mka|mkv|mov|mp3|mp4|mpeg|mpg|ogg|ogm|opus|otf|png|svg|svgz|swf|ttf|wav|webm|webp|woff|woff2|xml|zip)$`
	DefaultUseXForwardedHostValue         = false
)

// Client is used to interract with the DataDome's Protection API.
// This structure contains all the informations specified through the [Option]'s functions.
type Client struct {
	EnableGraphQLSupport      bool
	EnableReferrerRestoration bool
	Endpoint                  string
	Logger                    Logger
	MaximumBodySize           int
	ModuleName                string
	ModuleVersion             string
	ServerSideKey             string
	Timeout                   int
	UrlPatternInclusion       string
	UrlPatternExclusion       string
	UseXForwardedHost         bool

	endpoint            string
	httpClient          *http.Client
	urlPatternExclusion *regexp.Regexp
	urlPatternInclusion *regexp.Regexp
}

// OperationType describes the expected operations values for a GraphQL query.
type OperationType string

const (
	Mutation     OperationType = "mutation"
	Query        OperationType = "query"
	Subscription OperationType = "subscription"
)

// GraphQLData describes the informations extracted from the GraphQL query
type GraphQLData struct {
	Type  OperationType
	Name  string
	Count int
}

// ProtectionAPIRequestPayload is used to construct the payload that will be send to the Protection API
type ProtectionAPIRequestPayload struct {
	Key                    string        `url:"Key"`
	RequestModuleName      string        `url:"RequestModuleName"`
	ModuleVersion          string        `url:"ModuleVersion,omitempty"`
	ServerName             string        `url:"ServerName,omitempty"`
	APIConnectionState     string        `url:"APIConnectionState,omitempty"`
	IP                     string        `url:"IP"`
	Port                   string        `url:"Port,omitempty"`
	TimeRequest            string        `url:"TimeRequest,omitempty"`
	Protocol               string        `url:"Protocol,omitempty"`
	Method                 string        `url:"Method,omitempty"`
	ServerHostName         string        `url:"ServerHostname,omitempty"`
	Request                string        `url:"Request"`
	HeadersList            string        `url:"HeadersList,omitempty"`
	Host                   string        `url:"Host,omitempty"`
	UserAgent              string        `url:"UserAgent,omitempty"`
	Referer                string        `url:"Referer,omitempty"`
	Accept                 string        `url:"Accept,omitempty"`
	AcceptEncoding         string        `url:"AcceptEncoding,omitempty"`
	AcceptLanguage         string        `url:"AcceptLanguage,omitempty"`
	AcceptCharset          string        `url:"AcceptCharset,omitempty"`
	Origin                 string        `url:"Origin,omitempty"`
	XForwardedForIP        string        `url:"XForwardedForIP,omitempty"`
	XRequestedWith         string        `url:"X-Requested-With,omitempty"`
	Connection             string        `url:"Connection,omitempty"`
	Pragma                 string        `url:"Pragma,omitempty"`
	CacheControl           string        `url:"CacheControl,omitempty"`
	CookiesLen             string        `url:"CookiesLen,omitempty"`
	CookiesList            string        `url:"CookiesList,omitempty"`
	AuthorizationLen       string        `url:"AuthorizationLen,omitempty"`
	PostParamLen           string        `url:"PostParamLen,omitempty"`
	XRealIP                string        `url:"X-Real-IP,omitempty"`
	ClientID               string        `url:"ClientID,omitempty"`
	SecChDeviceMemory      string        `url:"SecCHDeviceMemory,omitempty"`
	SecChUA                string        `url:"SecCHUA,omitempty"`
	SecChUAArch            string        `url:"SecCHUAArch,omitempty"`
	SecChUAFullVersionList string        `url:"SecCHUAFullVersionList,omitempty"`
	SecChUAMobile          string        `url:"SecCHUAMobile,omitempty"`
	SecChUAModel           string        `url:"SecCHUAModel,omitempty"`
	SecChUAPlatform        string        `url:"SecCHUAPlatform,omitempty"`
	SecFetchDest           string        `url:"SecFetchDest,omitempty"`
	SecFetchMode           string        `url:"SecFetchMode,omitempty"`
	SecFetchSite           string        `url:"SecFetchSite,omitempty"`
	SecFetchUser           string        `url:"SecFetchUser,omitempty"`
	Via                    string        `url:"Via,omitempty"`
	From                   string        `url:"From,omitempty"`
	ContentType            string        `url:"ContentType,omitempty"`
	TrueClientIP           string        `url:"TrueClientIP,omitempty"`
	GraphQLOperationCount  string        `url:"GraphQLOperationCount,omitempty"`
	GraphQLOperationName   *string       `url:"GraphQLOperationName,omitempty"`
	GraphQLOperationType   OperationType `url:"GraphQLOperationType,omitempty"`
}
