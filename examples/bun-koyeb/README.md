# Global API auth with Unkey and Koyeb

## Overview

[Koyeb](https://www.koyeb.com?ref=unkey) is a developer-friendly serverless platform to deploy apps globally. Koyeb offers a fully managed environment to deploy any apps in seconds without managing any infrastructure. Koyeb supports any programming languages, frameworks, and tools to build your apps.

This example shows how to build an API using bun, secure it with unkey, and deploy it globally on Koyeb.


## Requirements

- [Bun](https://bun.sh/) installed
- An [Unkey](https://unkey.dev/app) account
- A [Koyeb](https://www.koyeb.com?ref=unkey) account

## Install

```bash
bun install
```

## Develop locally

```bash
bun run dev
```

## Test

```bash
curl http://localhost:8000 -H "Authorization: Bearer <KEY>"
```


## Deploy on Koyeb


[![Deploy to Koyeb](https://www.koyeb.com/static/images/deploy/button.svg)](https://app.koyeb.com/deploy?type=git&name=bun-unkey&service_type=web&ports=8000;http;/&env[UNKEY_ROOT_KEY]=<root_key>&env[UNKEY_API_ID]=<api_id>&repository=github.com/unkeyed/unkey&branch=main&workdir=examples/bun-koyeb&builder=dockerfile)

Replace the environment variable placeholders with real values from your Unkey [dashboard](https://unkey.dev/app).

Then hit the `Deploy` button.
Koyeb will deploy your app in your selected regions and provide a unique URL to access it, or you can configure your own custom domain.


Now that your app is deployed, you can test it:

```sh-session
curl -XPOST https://<YOUR_APP_NAME>-<YOUR_KOYEB_ORG>.koyeb.app -H "Authorization: Bearer <UNKEY_API_KEY>"
```

It should return a `200` status code and at least the following response, depending on your key settings:
```json
{
  "valid": true
}
```


### Manual configuration:

1. Create a [new project](https://app.koyeb.com/apps/new) on Koyeb.
2. Under the advanced section, add your `UNKEY_ROOT_KEY` and `UNKEY_API_ID` environment variables. You can find those in the [Unkey dashboard](https://unkey.dev/app).
3. Click on the `Deploy` button


## References

- [Bun Documentation](https://bun.sh/docs)
- [Koyeb Documentation](https://www.koyeb.com/docs)
- [Unkey Documentation](https://unkey.dev/docs/introduction)
