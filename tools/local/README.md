### Unkey local development setup guide

Run the following command to setup the local development environment:

```sh
pnpm local [options]
```

List of available options:

- `--service=<service>` : Specifies which part of the application to develop. The values are `dashboard`, `api`, or `www`

- `--skip-env`: Skips the environment setup prompt if specified.

#### Example

```sh
pnpm local --service=dashboard --skip-env
```
