# Solution: Adding Suture_ID Header to All Operator API Requests

## Problem Statement
The user attempted to add a `Suture_ID` header to all requests made by the MariaDB operator to the Kubernetes API server by modifying `pkg/http/transport.go`, but it still didn't work for all requests.

## Root Cause Analysis
The issue was that while `pkg/http/transport.go` contained the `HeadersTransport` that adds the `Suture_ID` header, it was **only used by the custom HTTP client** in `pkg/http/client.go` for making HTTP requests to services like MaxScale.

The operator communicates with the Kubernetes API server using clients created by:
1. `ctrl.NewManager()` - Creates the controller-runtime manager
2. `kubernetes.NewForConfig()` - Creates Kubernetes clientsets
3. `client.New()` - Creates controller-runtime clients
4. `discoverypkg.NewDiscoveryClientForConfig()` - Creates discovery clients

All these clients use `rest.Config` from the Kubernetes client-go library, which has its own transport layer. The `HeadersTransport` in `pkg/http/transport.go` was never used for these Kubernetes API requests.

## Solution Implementation

### 1. Created a Wrapper Function
Added `WrapRestConfigWithSutureID()` function in `pkg/http/transport.go` that wraps a `rest.Config` to inject the `Suture_ID` header into all requests:

```go
func WrapRestConfigWithSutureID(config *rest.Config) {
	if config == nil {
		return
	}
	
	// Set the WrapTransport function to add the Suture_ID header
	originalWrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		// If there's an existing wrapper, apply it first
		if originalWrap != nil {
			rt = originalWrap(rt)
		}
		// Then wrap with our Suture_ID transport
		return NewHeadersTransport(rt, map[string]string{})
	}
}
```

This function:
- Takes a `rest.Config` and modifies it in-place
- Preserves any existing `WrapTransport` function
- Wraps the transport with the existing `HeadersTransport` that adds the `Suture_ID` header

### 2. Updated All Entry Points
Applied the wrapper to all entry points where `rest.Config` is created:

#### Main Controller (`cmd/controller/main.go`)
```go
restConfig, err := ctrl.GetConfig()
// ... error handling ...
pkghttp.WrapRestConfigWithSutureID(restConfig)
```

#### Agent (`cmd/agent/main.go`)
```go
restConfig, err := ctrl.GetConfig()
// ... error handling ...
mdbhttp.WrapRestConfigWithSutureID(restConfig)
```

#### Init Container (`cmd/init/main.go`)
```go
restConfig, err := ctrl.GetConfig()
// ... error handling ...
mdbhttp.WrapRestConfigWithSutureID(restConfig)
```

#### Certificate Controller (`cmd/controller/cert_controller.go`)
```go
restConfig := ctrl.GetConfigOrDie()
pkghttp.WrapRestConfigWithSutureID(restConfig)
```

#### Webhook Server (`cmd/controller/webhook.go`)
```go
restConfig := ctrl.GetConfigOrDie()
pkghttp.WrapRestConfigWithSutureID(restConfig)
```

#### Discovery Client (`pkg/discovery/discovery.go`)
```go
config, err := ctrl.GetConfig()
// ... error handling ...
mdbhttp.WrapRestConfigWithSutureID(config)
```

### 3. Added Comprehensive Tests
Created `pkg/http/transport_test.go` with tests for:
- `HeadersTransport.RoundTrip()` with and without `SUTURE_ID` environment variable
- `HeadersTransport.RoundTrip()` with request body (verifies Content-Type and Accept headers)
- `WrapRestConfigWithSutureID()` with various config scenarios

All tests pass successfully.

## How It Works

1. When any component of the operator starts, it obtains a `rest.Config` from the controller-runtime
2. Before using the config to create clients, we call `WrapRestConfigWithSutureID()` on it
3. This modifies the config's `WrapTransport` function to wrap all HTTP transports with our `HeadersTransport`
4. The `HeadersTransport` intercepts every HTTP request and adds the `Suture_ID` header from the environment variable
5. All subsequent requests to the Kubernetes API server now include the `Suture_ID` header

## Benefits of This Approach

1. **Centralized**: The header injection logic remains in one place (`pkg/http/transport.go`)
2. **Reusable**: The same `HeadersTransport` is used for both custom HTTP clients and Kubernetes API clients
3. **Non-invasive**: We don't modify the core request handling logic, just wrap the transport layer
4. **Preserves existing behavior**: Any existing `WrapTransport` functions are preserved
5. **Testable**: The solution is easily testable with unit tests

## Files Modified

1. `pkg/http/transport.go` - Added `WrapRestConfigWithSutureID()` function
2. `pkg/http/transport_test.go` - Added comprehensive tests (new file)
3. `cmd/controller/main.go` - Applied wrapper to main controller
4. `cmd/agent/main.go` - Applied wrapper to agent
5. `cmd/init/main.go` - Applied wrapper to init container
6. `cmd/controller/cert_controller.go` - Applied wrapper to cert controller
7. `cmd/controller/webhook.go` - Applied wrapper to webhook server
8. `pkg/discovery/discovery.go` - Applied wrapper to discovery client
9. `.gitignore` - Added patterns to exclude compiled binaries

## Testing

All existing tests pass:
```
Ginkgo ran 24 suites in 12.841019429s
Test Suite Passed
```

New tests for transport functionality all pass with comprehensive coverage.

## Conclusion

The solution successfully ensures that the `Suture_ID` header is added to **all** requests made by the operator to the Kubernetes API server, which was the original goal. The approach is clean, maintainable, and tested.
