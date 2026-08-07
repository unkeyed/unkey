# @unkey/design

The design docs for Unkey's UI primitives and page patterns, built with
[Blume](https://useblume.dev). It imports `@unkey/ui` from `web/internal/ui`, so
the site always renders the components of the checkout it runs in.

## Run it

```bash
pnpm --dir=web/apps/design dev
```

The `dev` script reads `PORT` and `HOST`, so a proxy can place the site on its
own hostname.

`pnpm build` refuses to run while a dev server is up, because both write the
same generated `.blume/` runtime. To verify a build without stopping the server:

```bash
./node_modules/.bin/blume build --isolated
```

That writes to `.blume-verify/` and publishes nothing.

## Layout

```
blume.config.ts              site config: title, logo, theme, examples dir
theme.css                    tokens for the docs chrome
preview.css                  tokens injected into every preview iframe
docs/<section>/meta.ts       sidebar group title, icon, page order
docs/<section>/<name>.mdx    a page
examples/<name>/<example>.tsx  a live example, default export
```

A page's route is its path under `docs/`, so `docs/primitives/skeleton.mdx`
serves at `/primitives/skeleton`.

## Add a page

1. Create `docs/<section>/<name>.mdx`. Frontmatter `title` becomes the `h1` and
   `description` becomes the lead paragraph, so do not repeat either in the body.
2. Create `examples/<name>/basic.tsx`. It must have a **default export**. Add
   more examples as sibling files.
3. Reference each one with `<Component path="<name>/basic" />`. Blume renders
   that file as both the live preview and the code tab, so the two cannot drift.
4. Add the slug to the `pages` array in `docs/<section>/meta.ts`. A page left out
   still appears, but after the listed ones.

Blume provides Card, CardGroup, Steps, Tabs, Badge, FileTree, Accordion,
CodeGroup, TypeTable and more in any `.mdx` file with no import. See the
[component reference](https://useblume.dev/docs/content/components).

## Previews are iframes

Each example renders in its own iframe, which the docs stylesheet never reaches.
`preview.css` is the only source of component tokens, and it mirrors
`web/apps/dashboard/styles/tailwind.css` so a preview shows what the dashboard
shows. Two things follow:

- A class from `@unkey/ui` only gets a rule because `preview.css` declares an
  `@source` for `internal/ui`. New source paths belong there, not in `theme.css`.
- A new semantic token in the dashboard's `tailwind.css` has to be copied into
  `preview.css`, or previews render it as nothing.

The iframe is its own viewport at the width of the content column, which is
below the `md` breakpoint. A component with `md:` classes shows its narrow
layout in a preview.
