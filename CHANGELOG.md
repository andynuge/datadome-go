# DataDome Go Module

## v2.1.0 (2025-05-12)

- Add `UseXForwardedHost` setting to support `host` override via `X-Forwarded-Host` header

## v2.0.0 (2025-03-04)

### Breaking changes

- Rename `DataDomeStruct` structure to `Client`
- Remove `DataDome` prefix on fields of the `Client` structure
- Replace sub-packages with a single `modulego` package
- Update the `NewClient` signature to use the functional options pattern
- Handle configuration errors during the client's instantiation

### General changes

- Add support of 301/302 redirections returned by the Protection API
- Add `Logger` field to the `Client` structure
- Enhance code documentation

## v1.3.0 (2024-12-18)

- Add `EnableReferrerRestoration` field to enable the referrer restoration

## v1.2.0 (2O24-10-30)

- Add GraphQL support for POST requests
  - Add `EnableGraphQLSupport` field to enable GraphQL support
  - Add `MaximumBodySize` field to define maximum amount of data to read on GraphQL requests
- Add debug logs and enhance log outputs
  - Add `Debug` field to enable debug mode

## v1.1.2 (2024-08-27)

- Update `TimeRequest` value to a timestamp in microseconds without floating point to comply with the API contract
- Update inclusion/exclusion regex matching to apply to the complete URL, making configuration simpler and more secure
- Update default URL pattern exclusion regex to ensure consistent regex format across all platforms
- Update truncation limits for the data sent to the API Server

## v1.1.1 (2023-12-04)

- Fix hostname for DataDome endpoint

## v1.1.0 (2023-11-27)

- Use hostname for endpoint configuration

## v1.0.0 (2023-11-16)

- First release 
