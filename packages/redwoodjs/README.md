# RedwoodJS CLI Plugin for Unkey

Sets up Unkey for use in RedwoodJS projects.

## Setup Unkey

```bash
`yarn rw setup package @unkey/redwoodjs`
```

- Installs [Unkey SDK](https://unkey.dev/docs/libraries/ts/sdk)
- Adds a client lib `unkey.ts` in your RedwoodJS `api/lib` directory
- Adds placeholder [Unkey Root Key](https://unkey.dev/docs/libraries/ts/sdk/overview#unkey-root-key) environment variable in your RedwoodJS `.env`

## Unkey CLI Commands

Run available RedwoodJS Unkey CLI commands via:

`yarn rw @unkey <command> <arg>`

For example,

`yarn rw @unkey setRootKey -root-key=<YOUR-UNKEY-ROOT-KEY>`

### Help

To get help or a list of available commands:

`yarn rw @unkey --help`

## Contributing

### Test in a local project

- cd `packages/redwoodjs`
- `pnpm build`
- run `setup.js` via:

```bash
node dist/setup.js --cwd=/path/to/your/redwoodjs/project
```

## TODO

- install the envelop plugin once published
- codemod the GraphQL handler to add plugin or better way of adding plugin
