---
name: refreshing-docs-screenshots
description: "Checks and refreshes product documentation screenshots from the real local dashboard using DashboardScreenshot declarations and data-docs-target markers. Use when asked to audit all docs screenshots, refresh a specific screenshot, or perform a monthly screenshot check."
argument-hint: "all | specific <target-or-src>"
---

# Refreshing docs screenshots

Read the screenshot descriptions in product docs, reproduce their states in the
real local dashboard, and replace only images that need an update. Use AI to
choose seed data and browser actions. Do not require fixed scenario scripts or
build a substitute UI.

## Choose a mode

Use one of these two modes:

- `all`: Find every `DashboardScreenshot` declaration under `docs/product/`.
  Check each image against the current dashboard and its surrounding docs.
  Refresh outdated or missing images. Leave accurate images unchanged.
  Report every declaration as current, refreshed, or blocked.
- `specific <target-or-src>`: Match a root-relative `src` path, or a `target`
  name if it identifies exactly one declaration. Check and, if needed, refresh
  only that screenshot and its theme variants. If several declarations use the
  same target, ask for the `src`. If a page contains several declarations, ask
  which image they mean. Stop on missing or duplicate matches.

Example requests:

> Use refreshing-docs-screenshots in all mode to check the product docs.

> Use refreshing-docs-screenshots in specific mode for root-key-permissions.

A monthly run can use `all`. Do not create a schedule unless the user explicitly
asks for one. An audit or refresh request does not authorize a push or PR.

## Read the declarations

Read `docs/product/snippets/dashboard-screenshot.jsx` before interpreting its
props. Discover declarations with a scoped search, then read the MDX and nearby
text. Ignore component names inside explanatory code blocks. Read capture
metadata from the MDX source, not the rendered docs HTML. The docs component
only renders images; it does not emit capture metadata or execute the workflow.

```bash
rg -n '<DashboardScreenshot' docs/product --glob '*.mdx'
```

The declaration's contract is:

- `target`: Value of `data-docs-target` on an existing dashboard element.
- `description`: Desired data, UI state, navigation hints, and capture constraints.
- `capture`: `target`, `viewport`, or `full-page`. Use `target` when omitted.
- `src`: Root-relative image path without the theme suffix or extension. The
  component renders `${src}-light.png` and `${src}-dark.png`. This identifies
  the saved illustration; no separate `id` is needed.
- `alt`: Reader-facing description of the image.
- `width`: Optional maximum display width in CSS pixels, not a capture viewport.
- `capturedAt`: ISO 8601 UTC timestamp for the saved capture, for example
  `2026-09-11T04:48:23Z`. It stays in the MDX source, not the docs DOM.

Use `capturedAt` to prioritize old or unknown captures. An image older than about
a month deserves attention, but age alone does not make it outdated. Recent
captures can also become wrong after UI changes. In `all` mode, account for every
declaration regardless of age. Treat missing, invalid, or future timestamps as
unknown, not fresh.

The timestamp records capture time, not the last audit. Never set it from the
render clock or advance it merely because an old image still looks correct.
Only update it after both theme images have been captured, inspected, and saved.
Do not invent a timestamp for an existing image with unknown capture history.

Plain `<img>` screenshots have no capture instructions. Report them as outside
this declaration-based workflow when relevant; do not silently claim they were
checked or convert them all without a request.

## Prepare the real dashboard

Read repository guidance and
`docs/engineering/contributing/local/development.mdx`. Use `mise` for tools.
Inspect running services before starting anything. Reuse a healthy local stack.

- On a developer machine, `mise run dashboard` is the dashboard setup task.
  Read its effects before running it, especially its database seeding step.
- In an orb, use the declared `.amp/services.yaml` services and supervised
  service commands. Load `using-agent-browser` before browser work. Share portal
  URLs with the user, not localhost URLs.
- Confirm local authentication and disposable local database connections before
  writing data. Never dump credentials or environment files into the transcript.

Use synthetic fixtures and create only the data needed by the description.
Prefer existing local seed helpers or the dashboard's normal creation flows.
Do not use production accounts, shared databases, real customer data, or real
credentials. Stop dependent work if a safe local environment is unavailable.

Locate the target in `web/apps/dashboard/`. Use the existing route and UI.
Do not create a preview component, mock dashboard, or new route to make capture
easier. If a marker is missing, report it as blocked unless the user authorized
adding it. An authorized marker belongs on an existing DOM container or a
component that forwards it to that container, including portaled dialog content.

## Check and capture

For each declaration:

1. Inspect the saved images and the surrounding docs. Identify what the image
   must teach. Use source changes as supporting evidence, not a substitute for
   checking the rendered UI.
2. Navigate the actual dashboard to the described state. Follow the description,
   but adapt navigation when the UI changes. Do not create a root key or perform
   another sensitive action if opening its form is sufficient.
3. Apply any declared viewport and locale. Otherwise use a 1440 by 1000 CSS-pixel
   viewport and record that choice. Capture at device scale 2. Wait for data,
   fonts, and opening animations to finish; a successful click is not readiness.
4. Find `[data-docs-target="<target>"]` in the dashboard document. Require
   exactly one visible match with nonzero bounds. Missing or ambiguous targets
   block capture. The docs page does not contain these markers.
5. For `target`, capture that element. For `viewport`, capture the current
   viewport after confirming the target identifies the expected state. For
   `full-page`, capture the document's scrollable page. A full-page capture does
   not expand independently scrolling panels. Follow browser-tool guidance for
   scrolled element crops; never accept a blank or truncated result.
6. Compare the current UI with the saved image. Refresh for changed controls,
   layout, labels, missing content, or a mismatch with the docs description.
   Different synthetic names or dates alone need not cause a replacement unless
   those values matter to the explanation. If the image remains accurate, leave
   it and its `capturedAt` unchanged.
7. Capture both light and dark variants before replacing either. Switch themes
   through the app or its supported system-theme behavior. Do not recolor an
   image or alter application CSS to fabricate a state.
8. Inspect both captures with the media tool. Check the intended content, crop,
   readability, theme, and absence of secrets. Keep the existing pair if either
   capture fails. Do not make docs prose agree with a wrong screenshot.
9. Save the verified pair at the paths derived from `src` under `docs/product/`.
   Keep paths within that directory. Set `capturedAt` to the UTC time the pair
   was captured, not when a later audit runs. Keep output paths stable.

Save review images in `.amp/in/artifacts/` in an orb. Keep temporary comparison
files elsewhere and remove them when finished. Do not use image generation for
dashboard screenshots.

## Verify and report

Load `docs-writing` for documentation changes. Use the Mintlify version pinned
in `docs/product/Dockerfile`; run it through `mise exec`. Run `mintlify validate`
from `docs/product/` after changes. Preview the affected real product docs page,
check both theme images load, and inspect the rendered result. Avoid enlarging
a narrow crop beyond its original CSS width; use `width` when needed.

Run scoped formatting or code checks if source changed, and `git diff --check`.
Preserve unrelated work. Close browser sessions and clean up only resources
created for this run. Do not stop a teammate's development stack.

Report the selected mode, image paths checked, current/refreshed/blocked results,
reasons for replacements, capture timestamps, verification, and any coverage gaps.
Include an inspected representative image or a real docs preview link.

Leave changes local unless the user authorizes publishing. If asked to open a
PR, load `creating-pull-requests` and create a draft with the before/after images,
reasons, and verification results. This skill does not grant permission to push,
open a PR, or schedule future runs.
