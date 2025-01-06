# DataDome Go Module

## 1.3.0 (2024-12-18)

- Add `EnableReferrerRestoration` field to enable the referrer restoration

## 1.2.0 (2O24-10-30)

- Add GraphQL support for POST requests
  - Add `EnableGraphQLSupport` field to enable GraphQL support
  - Add `MaximumBodySize` field to define maximum amount of data to read on GraphQL requests
- Add debug logs and enhance log outputs
  - Add `Debug` field to enable debug mode

## 1.1.2 (2024-08-27)

- Update `TimeRequest` value to a timestamp in microseconds without floating point to comply with the API contract
- Update inclusion/exclusion regex matching to apply to the complete URL, making configuration simpler and more secure
- Update default URL pattern exclusion regex to ensure consistent regex format across all platforms
- Update truncation limits for the data sent to the API Server

## 1.1.1 (2023-12-04)

- Fix hostname for DataDome endpoint

## 1.1.0 (2023-11-27)

- Use hostname for endpoint configuration

## 1.0.0 (2023-11-16)

- First release 
