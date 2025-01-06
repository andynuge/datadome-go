package models

type DataDome struct {
	DatadomeServerSideKey     string
	DataDomeEndpoint          string
	DataDomeTimeout           int
	EnableGraphQLSupport      bool
	EnableReferrerRestoration bool
	UrlPatternInclusion       string
	UrlPatternExclusion       string
	MaximumBodySize           int
	ModuleName                string
	ModuleVersion             string
	Debug                     bool
}

type GraphQLOperationType string

const (
	Mutation     GraphQLOperationType = "mutation"
	Query        GraphQLOperationType = "query"
	Subscription GraphQLOperationType = "subscription"
)

type GraphQLData struct {
	Type  GraphQLOperationType
	Name  string
	Count int
}

type DataDomeRequest struct {
	Key                    string
	RequestModuleName      string               `url:"RequestModuleName,omitempty"`
	ModuleVersion          string               `url:"ModuleVersion,omitempty"`
	ServerName             string               `url:"ServerName,omitempty"`
	APIConnectionState     string               `url:"APIConnectionState,omitempty"`
	IP                     string               `url:"IP,omitempty"`
	Port                   string               `url:"Port,omitempty"`
	TimeRequest            string               `url:"TimeRequest,omitempty"`
	Protocol               string               `url:"Protocol,omitempty"`
	Method                 string               `url:"Method,omitempty"`
	ServerHostName         string               `url:"ServerHostname,omitempty"`
	Request                string               `url:"Request,omitempty"`
	HeadersList            string               `url:"HeadersList,omitempty"`
	Host                   string               `url:"Host,omitempty"`
	UserAgent              string               `url:"UserAgent,omitempty"`
	Referer                string               `url:"Referer,omitempty"`
	Accept                 string               `url:"Accept,omitempty"`
	AcceptEncoding         string               `url:"AcceptEncoding,omitempty"`
	AcceptLanguage         string               `url:"AcceptLanguage,omitempty"`
	AcceptCharset          string               `url:"AcceptCharset,omitempty"`
	Origin                 string               `url:"Origin,omitempty"`
	XForwardedForIP        string               `url:"XForwardedForIP,omitempty"`
	XRequestedWith         string               `url:"X-Requested-With,omitempty"`
	Connection             string               `url:"Connection,omitempty"`
	Pragma                 string               `url:"Pragma,omitempty"`
	CacheControl           string               `url:"CacheControl,omitempty"`
	CookiesLen             string               `url:"CookiesLen,omitempty"`
	AuthorizationLen       string               `url:"AuthorizationLen,omitempty"`
	PostParamLen           string               `url:"PostParamLen,omitempty"`
	XRealIP                string               `url:"X-Real-IP,omitempty"`
	ClientID               string               `url:"ClientID,omitempty"`
	SecChDeviceMemory      string               `url:"SecCHDeviceMemory,omitempty"`
	SecChUA                string               `url:"SecCHUA,omitempty"`
	SecChUAArch            string               `url:"SecCHUAArch,omitempty"`
	SecChUAFullVersionList string               `url:"SecCHUAFullVersionList,omitempty"`
	SecChUAMobile          string               `url:"SecCHUAMobile,omitempty"`
	SecChUAModel           string               `url:"SecCHUAModel,omitempty"`
	SecChUAPlatform        string               `url:"SecCHUAPlatform,omitempty"`
	SecFetchDest           string               `url:"SecFetchDest,omitempty"`
	SecFetchMode           string               `url:"SecFetchMode,omitempty"`
	SecFetchSite           string               `url:"SecFetchSite,omitempty"`
	SecFetchUser           string               `url:"SecFetchUser,omitempty"`
	Via                    string               `url:"Via,omitempty"`
	From                   string               `url:"From,omitempty"`
	ContentType            string               `url:"ContentType,omitempty"`
	TrueClientIP           string               `url:"TrueClientIP,omitempty"`
	GraphQLOperationCount  string               `url:"GraphQLOperationCount,omitempty"`
	GraphQLOperationName   *string              `url:"GraphQLOperationName,omitempty"`
	GraphQLOperationType   GraphQLOperationType `url:"GraphQLOperationType,omitempty"`
}
