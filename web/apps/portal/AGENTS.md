# Portal

Customer-facing portal: an end user of an Unkey customer reaches it through a
short-lived session cookien. TanStack Start, TanStack Router file
routes, React 19, react-query, Tailwind v4, base-ui primitives.

The repo-level `AGENTS.md` applies. This file only adds what is portal-specific.

## Where things go

```
src/
  routes/          route config only: validateSearch, beforeLoad, component. No JSX.
  components/
    <page>/        one folder per page: the render-only page component, its pure
                   model with tests, and its named state components (error, empty)
    <feature>/     presentational components and pure helpers for one feature
    ui/            base-ui primitives (shadcn-style), no app logic
  hooks/           every custom hook in the app, one per file, `use-<name>.ts`
  lib/             server functions, session, scopes, env, db. No React.
  styles/          tailwind.css and the portal design tokens
```

Hooks live in `src/hooks/`, page controllers, query hooks, mutation hooks and DOM hooks.

Pure derivation lives next to the component that renders it, as
`<name>-model.ts` or a plain `.ts`, and never imports from a `.tsx` file.

## Page shape

A page is four layers, each importing only downward:

1. `routes/_portal/<page>.tsx`: route config, `component: <Page>`.
2. `hooks/use-<page>.ts`: the controller. Composes search state, data hooks and
   one pure derive call, returns a discriminated view model.
3. `components/<page>/<page>-model.ts`: pure. Owns every derivation and the
   tests.
4. `components/<page>/<page>.tsx`: render only. One branch on `view.status`,
   then destructure and lay out.

Fetching sits behind one hook per page (`hooks/use-<page>-usage.ts` or
similar) that returns a single typed object.

## Rules

- Try not to use `useEffect` in components. Derive it, handle it in the event, or put it in
  a hook. `hooks/use-mount-effect.ts` covers mount-only subscriptions.
- No boolean props that decide whether a section exists. Pass an optional
  object and test presence.
- Query and mutation hooks own their side effects: invalidation, reset,
  optimistic state. Call sites destructure and render.
- Dialog flows get one hook (e.g `hooks/use-rotate-key.ts`): which
  item is open, the form state, the mutation, open and close.
- Table state (sort, page, page size) lives in a hook, reset by remounting the
  table with a `key` computed once in the model.
- Scopes gate features through `lib/scopes.ts`.

## Session

The session is a react-query entry, `sessionQueryOptions` in `lib/session.ts`,
held by the router's query client. `_portal.tsx` reads it in `beforeLoad` with
`ensureQueryData`; the not-found page reads it with `useQuery`; the entry route
refetches it after a code exchange. Any query failing with a 401 removes the
entry through `QueryCache.onError` in `router.tsx`. Server functions that can
401 throw `SESSION_EXPIRED_MESSAGE` from `lib/portal-api.ts`.

## Verification

```bash
mise exec -- pnpm --dir=web/apps/portal exec tsc --noEmit
mise exec -- pnpm --dir=web/apps/portal exec biome check src
mise exec -- pnpm --dir=web/apps/portal exec vitest run
mise exec -- pnpm --dir=web/apps/portal build
```

`src/routeTree.gen.ts` is generated and gitignored. After adding or deleting a
route, run the build before `tsc` or the types lag behind the tree.


