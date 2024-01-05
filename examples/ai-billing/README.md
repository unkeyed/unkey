# How to use Unkey to set up billing for your AI application

Example of a generative AI application using Unkey for billing.

- Includes example code to set up Stripe for payment links
- On payment, users are assigned an Unkey API key with the 'remaining' field set to 10, signifying 10 credits
- This API key is saved to a cookie (httpOnly, so not accessible via client-side Javascript)
- This cookie is attached to requests to an API route in /api/openai; this API route verifies the key (decrementing `remaining`) and requests images from OpenAI

### Deploy to Vercel

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Funkeyed%2Fhackbill&env=AUTH_SECRET,AUTH_GITHUB_ID,AUTH_GITHUB_SECRET,UNKEY_ROOT_KEY,UNKEY_API_ID,OPENAI_API_KEY,STRIPE_SECRET_KEY,STRIPE_PUBLISHABLE_KEY,STRIPE_PRICE_ID&demo-title=Billing%20%26%20Analytics%20with%20Unkey&demo-url=https%3A%2F%2Fhackbill.vercel.app%2F&integration-ids=oac_pBxhD462CCUdby4F6vTacGrd&skippable-integrations=1)
