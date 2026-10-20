# Deploy demo

Keep this a small, stateless Zen API that runs without secrets or other services.
The Unkey gateway owns API key verification. Do not add key creation endpoints,
echo request credentials or environment variables, or trust caller-supplied
principal headers. Keep the root and health routes outside the `/api/` policy.
