<div align="center">
    <h1 align="center">Workflows</h1>
    <h5>Durable workflows powered by  <a href="https://trigger.dev">trigger.dev</a></h5>
</div>
<br/>

## Getting Started

### Development

1. Create a `.env` file next to `.env.example`

```bash
cp .env.example .env
```

2. Populate the `.env` file with all required variables

3. Run the Next.js app

```bash
pnpm run dev
```
Or if you have infisical access:
```bash
infisical run --env=dev -- pnpm dev
```

4. In a different terminal, run the trigger tunnel

```bash
pnpm dlx @trigger.dev/cli@latest dev
```

5. Go to Trigger.dev, where you can test the runs
