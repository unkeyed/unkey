# unkey-cli

CLI login example built with [Unkey](https://unkey.dev/)

# Running this locally

This template involves a deployed Next.js web app in `examples/nextjs` and a command-line script in `/src/index.ts.`. To run through the auth flow locally:

1. Install dependencies
2. Run the script in dev mode from the monorepo root: `pnpm dev login`
3. This will open the web app in your browser, where you can confirm or cancel the auth flow

# About

## How this works

1. CLI spins up server on localhost
2. CLI opens a page in the user's browser (/auth/devices) which is protected by an auth solution (Clerk, in this case)
3. The user verifies that the code there matches the one shown in their terminal
4. Upon verification, the Next.js app mints a new Unkey API key and sends it back to the local server
5. The script writes the API key to a file and kills the server

See the diagram below:

[diagram]

## Login

`unkey-cli login` does the following:

1. Opens a new browser window for your user to log in with your service
2. If this is sucessful, queries Unkey for a new API key
3. Stores the key locally in `.unkey`

## Example use case

[Vercel](https://vercel.com) has a CLI for deploying web applications (as well as a their more frequently used web app)

If you want to deploy your frontend with their CLI, you:

1. Run `vercel login` to log in to the web app and set a local credential in `.vercel`
2. Run `vercel deploy` which takes local files and uploads them to the Vercel CI/CD to be deployed.

This is a simplified example the kind of tool that you could build with this template.
