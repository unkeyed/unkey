# RedwoodJS CLI Plugin for Unkey

Sets up Unkey for use in RedwoodJS projects.

## Setup Unkey

```bash
`yarn rw setup package @unkey/redwoodjs`
```

- Installs Unkey SDK
- Adds a client lib `unkey.ts` in your RedwoodJS `api/lib`` directory

## Run a command

If we have some Unkey specific function/feature generator commands

`yarn rw @unkey <command> <arg>`

`yarn rw @unkey set-token <token>`

## Contributing

### Test in a local project

- cd `packages/redwoodjs`
- `pnpm build`
- run `setup.js` via:

```bash
node dist/setup.js --cwd=/path/to/your/redwoodjs/project
```

## TODO

- sets dummy envar token

- install the envelop plugin once published
- codemod the GraphQL handler to add plugin or better way of adding plugin
