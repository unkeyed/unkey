Todos for Demo

- [x] Basic HTTP Server that proxies requests

  - Forwards request to some VM from the db
  - Forwards the request body
  - Forwards the request headers

- [] Handles HTTPs requests with SSL certificate

  - [] Gateway handles SSL letsencrypt challenges HTTP-01
  - [] Handle local generated certificate `mkcert api.unkey.app`
  - [x] Handles actual SSL cert pointing to localhost

- [x] Hook up VM's to control-plane

  - [] Generate gateway config
  - [] Insert the VM's into the db
  - [] Update's vm status in the db
  - [] Remove go/cmd/seed-gateway

- [] Error handling

  - Implement correct gateway status codes e.g 502 503 504
  - Ensure requests timeout after a certain duration e.g 30 seconds
    - Demo endpoint for testing error handling

- [] API key verify

  - keyspaceid
  - use key service from go/internal/services/keys/get.go
  - cli json

- [] OpenAPI request validation
- just use current validator

<!--- Find good library that supports spec 2.0 3.0 3.1
- or write our own
- Make this opt in as some API's might auto generate the schema and validate them ourselves-->

[] Make it work on the CLOUD!

AFTER DEMO:

- [] Ratelimiting

  - Calls ratelimit api?
  - or does it have the ratelimit service?

- [] Request logging

  - Log requests into clickhouse
  - Some sort of config to determine what of the request and or response we should redacted, since we would potentially log senstive information.
  - Potentially flag to turn off logging via gateway config / response header

- [] Adjust cache lib

  - Add SWR Many / getmany fun
