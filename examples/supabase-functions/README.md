# Supabase functions secured with Unkey

This examples shows you, how you can secure your Supabase functions with Unkey.

## Usage

1. Go to [unkey.dev](https://unkey.dev/app) and create an account, make sure you have an account with Supabase.

2. Run the example
```bash
supabase start
supabase functions serve
```
3. Create an API Key in Unkey in the Dashboard or API

4. Access the Hello world function via CURL.

```bash
curl -XPOST -H 'Authorization: Bearer <SUPABASE_FUNCTION_JWT' -H 'x-unkey-api-key: <API_KEY_FROM_UNKEY' -H "Content-type: application/json" 'http://localhost:54321/functions/v1/hello-world'
```

- [Getting Started](https://docs.unkey.dev/quickstart) - A quickstart guide to Unkey