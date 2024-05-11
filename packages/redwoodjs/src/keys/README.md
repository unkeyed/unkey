# RedwoodJS Api Key Verification

`createApiKeyMiddleware` is middleware function for RedwoodJS used to validate API keys sent in the request headers with Unkey.

In the future, support for key verification in GraphQL operations and other RedwoodJS functions will be added.

## Usage

Here's a basic example of how to use `createApiKeyMiddleware`:

```javascript

```

In this example, `createApiKeyMiddleware` is used as a global middleware. It will validate the API key for all incoming requests.

## Configuration

`createApiKeyMiddleware` can be configured by passing an options object to the function. Here's an example:

```javascript

```

In this example, `createApiKeyMiddleware` will look for the API key in the 'X-API-KEY' header and validate it by comparing it to 'expected-key'.

## Error Handling

If the API key is missing or invalid, `createApiKeyMiddleware` will send a 401 Unauthorized response and stop the request from being processed further. You can customize this behavior by providing your own error handling function in the options object:

```javascript

```
