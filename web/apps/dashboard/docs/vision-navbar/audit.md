# Dashboard navigation audit

> Phase 1 of [DES-32 — Update the navbar](https://linear.app/unkey/issue/DES-32/update-the-navbar), under [DES-29 — Vision: Dashboard prototype](https://linear.app/unkey/issue/DES-29/vision-dashboard-prototype).
>
> A map of what the dashboard's navigation chrome looks like today, why each piece exists, and the problems that need solving before we explore variations. **No designs in this doc** — this is the brief, not the answer.

## Why we're doing this

Three threads converged.

- **Apr 1 (Andreas / Dave):** "We don't really have someone who really owns that space." The product switcher in the top-left is "a temporary fix." Nav/IA is a known open problem. The dashboard "looks nothing like the new marketing site."
- **Apr 7 (Oz / Dave):** "API management and Deploy sections look like two different apps." Component library is partial. No one is preventing one-off UI implementations.
- **Apr 17 (Andreas / Dave sync):** Dave's first week surfaced many interdependent UI issues that can't be resolved piecemeal. → DES-29 was opened.

Dave's Slack note (Apr 20) scoped the work down: navbar first, defer styling. This audit narrows further — first **understand the current state**, then explore variations.

## Scope: four chrome zones

| # | Zone | Today's owner |
| --- | --- | --- |
| 1 | Sidebar (left) | `AppSidebar` |
| 2 | Top breadcrumb bar (incl. action slot, sidebar collapse, user menu) | `Navbar` + per-section assemblers — **mounted inconsistently** |
| 3 | In-page section header (title + description + primary action) | None — every page rolls its own |
| 4 | Project-level action slot (Redeploy, Create deployment, Rollback) | Hardcoded inside `ProjectNavigation`, sitting in zone 2 |

**The pattern that jumps out:** zones 3 and 4 have no shared owner. Each section invents its own header and its own action placement. That's the structural cause of Oz's "two different apps" feeling — and it's why this audit needs to cover all four zones, not just the sidebar.

---

## Zone 1 — Sidebar

### Architecture

- **Layout entry** — `app/(app)/layout.tsx:67–113` mounts `SidebarProvider`, then `AppSidebar` (desktop) and `SidebarMobile` (mobile sheet, separate component).
- **AppSidebar** — `components/navigation/sidebar/app-sidebar/index.tsx:42–217`. Three vertical regions:
  - **Header** (`SidebarHeader`, lines 113–142): `ProductSwitcher` + a separate sidebar-collapse button.
  - **Content** (`SidebarContent`, lines 148–199): collapsed-state toggle + back button (lines 151–180) → `ContextNavigation` (the live nav for the current product/resource) → `WorkspaceSection` (Audit / Customer Billing (beta) / Settings) → `UsageBanner` at the bottom.
  - **Footer** (`SidebarFooter`, lines 200–213): `WorkspaceSwitcher` + `HelpButton`.
- **Active state** is derived in `hooks/use-navigation-context.ts:48–155` from route segments + a `localStorage` fallback (the saved product).
- **Nav data** is static config in `components/navigation/sidebar/navigation-configs.tsx` — separate functions for API Management (`createApiManagementNavigation`, lines 35–84) and Deploy (`createDeployNavigation`, lines 89–100), plus per-resource navs for API / Project / Namespace.
- **Product switcher** — `components/navigation/sidebar/product-switcher.tsx:30–158`. Hardcoded two-product list (lines 38–51). Calls `switchProduct` from `useProductSelection`, which writes to `localStorage` and dispatches a custom `STORAGE_EVENT` so other components re-read.
- **Workspace switcher** — `components/navigation/sidebar/team-switcher.tsx:32–221`. Lives at the bottom of the sidebar. Triggers `trpc.user.switchOrg` and full-page reloads via `window.location.replace("/")` (lines 54–84).
- **Workspace section** — `components/navigation/sidebar/workspace-section.tsx:15–84`. Static items list with `hidden` filtering on line 71 — Customer Billing only renders if `workspace.betaFeatures?.portal`.

### Problems

- **Product switcher is explicitly temporary.** Andreas (Apr 1): *"the product switcher … gets the job done. But like I don't want to just get the job done."* It exists because Deploy got bolted on next to API Management with no IA story.
- **Workspace switcher and product switcher are spatially disjoint.** Product is top-left of the sidebar header; workspace is bottom-left of the sidebar footer. They're conceptually peers (both "what am I in?") but visually unrelated. New users have to discover them as two separate interactions.
- **Sidebar mixes three different scopes** with no visual hierarchy:
  - Workspace-level (Audit, Settings, Customer Billing)
  - Product-level (APIs, Ratelimit, Authorization, Logs, Identities — or — Projects)
  - Resource-level (Requests, Keys, Settings under a specific API; Deployments / Logs / etc. under a specific Project)
  - All three live in the same scrollable column, separated only by a `border-t` divider (`app-sidebar/index.tsx:188`).
- **Beta-gated items appear/disappear with no signposting.** `betaFeatures.portal` flips Customer Billing in and out of the nav (`workspace-section.tsx:37`). Users have no way to know what's hidden.
- **Mobile sidebar is a separate component.** `SidebarMobile` lives at `components/navigation/sidebar/sidebar-mobile.tsx`. Any nav structure change has to be made twice.
- **Active-state derivation is fragile.** `use-navigation-context.ts:74–95` does string `startsWith` matching on segment names (e.g., `productSegment.startsWith("project")`). Any new top-level route requires updating this list or it falls into the wrong product context.

---

## Zone 2 — Top breadcrumb bar

### Architecture

- **Component** — `components/navigation/navbar.tsx:149–178`. `Navbar` is a compound component:
  - `Navbar.Breadcrumbs` (lines 107–144) — left side, takes a leading icon + breadcrumb links.
  - `Navbar.Actions` (lines 43–49) — right-of-spacer slot for arbitrary buttons.
  - `Navbar.User` (lines 52–58) — avatar / user menu, defaulted to `<UserButton />` if not provided. Hidden below `md` breakpoint.
- **Breadcrumb dropdowns** ("Local API ⌄") — each breadcrumb crumb can render a `QuickNavPopover` (`components/navbar-popover.tsx:42–239`) with virtualized list, keyboard shortcuts (option+shift+key), and active-item highlighting.
- **Per-section assemblers** build the breadcrumb config:
  - Project routes: `app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/navigations/use-breadcrumb-config.ts:31–165` returns a declarative array; sub-pages list is hardcoded (lines 50–93).
  - Other sections inline their own simpler breadcrumbs.
- **`UserButton`** — `components/navigation/sidebar/user-button.tsx:28–93`. Note the file path: it lives under `sidebar/` even though it now renders inside the navbar. Theme switcher + sign out.

### Mount points (this is the load-bearing problem)

There is **no global mount** of `Navbar`. Each section decides for itself whether to render one. Catalog of who mounts what:

| Route | Navbar mounted? | Where |
| --- | --- | --- |
| `apis/page.tsx` | Yes — inline in the page itself | `apis/page.tsx:19–28` |
| `projects/` (list) | Yes — via `ProjectsListNavigation` | `projects/navigation.tsx:8–27` |
| `projects/[projectId]/...` | Yes — via `ProjectNavigation` mounted by the `[projectId]/layout.tsx` | `projects/[projectId]/layout.tsx:37–42` |
| Workspace root, settings, audit, logs, identities, ratelimits | **No** | — |

### Problems

- **Inconsistent mounting** is the structural problem. Two of three patterns mount it from a page (not a layout), one from a layout. Routes that don't mount it have no breadcrumbs at all, which means the user has no orient-by-trail in roughly half the dashboard.
- **Each mount point reinvents the breadcrumb config.** Projects has a declarative `useBreadcrumbConfig`. APIs hard-codes one link inline. Projects-list hard-codes one link inline. There's no single source of truth.
- **`UserButton` lives under `sidebar/`** even though it renders in the top bar. Cleanup signal — the chrome wasn't designed top-down.
- **No room for in-app notifications.** Dave's [FigJam sticky](https://www.figma.com/board/bWIYSBO9hPnPcfvBJNy6U4/Deploy--Navbar-Exploration?node-id=2004-168): *"I have a feeling at some point, you guys are going to need in-app notifications for stuff. e.g changelog updates, build failures, dependency updates."* The current right-side slot is taken by `Navbar.Actions` (which is per-section), the sidebar collapse, and the avatar. Nowhere to put a global notifications bell.
- **The sidebar collapse button lives in two places.** Once in the sidebar header (`app-sidebar/index.tsx:124–128`), and again as `<DoubleChevronLeft>` in `ProjectNavigation` (`project-navigation.tsx:137–145`) where it actually toggles a *details panel*, not the sidebar. Same icon, different behavior — confusing.
- **Cloudflare / Vercel cautionary tales** (Apr 1). Andreas on Vercel's recent redesign: *"the top level, you change in the sidebar. Then you go to the top nav bar … then you go back to the sidebar where the third level is."* We need to not do that.

---

## Zone 3 — In-page section headers

### Architecture

There is **no shared `<PageHeader>`**. Each page renders its own title + description + action row. Sample:

| File | Pattern |
| --- | --- |
| `app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/components/deployments-header.tsx:7–26` | `flex flex-col md:flex-row` with `<h1 className="font-semibold text-gray-12 text-lg leading-8">` + description + `<CreateDeploymentButton renderTrigger={...}>` |
| `app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/env-vars/components/toolbar/env-vars-header.tsx:11–26` | `flex items-start justify-between` with same h1/description shape + a state-driven `<Button variant={isAddOpen ? "outline" : "primary"}>` |
| `app/(app)/[workspaceSlug]/apis/_components/create-api-button.tsx:33–115` | Different pattern: button is a `NavbarActionButton` (zone 2), and there's no in-page header for the APIs list at all |

### Problems

- **No source of truth.** Every section reinvents the title + description + action layout. Drift is inevitable — and is exactly what Oz called out: *"API management and Deploy sections look like two different apps."*
- **Title typography is duplicated literal-for-literal** (`font-semibold text-gray-12 text-lg leading-8`) across multiple files. A `<PageTitle>` primitive would let one change touch every page.
- **The "primary action" placement is ambiguous.** Sometimes it lives in the in-page header (`DeploymentsHeader`'s "Create Deployment"); sometimes it lives in the navbar action slot (`apis/page.tsx`'s "Create new API"). The same affordance has two different homes depending on the section.

---

## Zone 4 — Project-level action slot

### Architecture

`app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/navigations/project-navigation.tsx:98–149` — a `renderActions()` function returning a row of buttons inside `Navbar.Actions`:

- `<CreateDeploymentButton>` (lines 102–110) — only shown if `activeProject?.repositoryFullName` is set.
- **Redeploy** button (lines 111–124) — wraps `<ArrowDottedRotateAnticlockwise>` in an `InfoTooltip` (*"Redeploy the current active deployment"*). Disabled if there's no `selectedDeployment`. Opens `RedeployDialog` (lines 125–131).
- **Details-panel toggle** (lines 132–145) — `<DoubleChevronLeft>` button wired to the `onClick` prop, which toggles `isDetailsOpen` on `ProjectLayoutContext`. Tooltip flips between "Show / Hide deployment details" and "No deployments available."

The dialog itself: `app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/components/table/components/actions/redeploy-dialog.tsx`.

The layout context that owns the details-panel state: `app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/layout-provider.tsx:1–22`.

### Problems — the debate that motivated this audit

Dave (Apr 20): *"I want to come up with patterns that make sense for these types of 'project level' actions like 'Redeploy the current active deployment'. There's been a lot of debate about whether these belong in this section. We need new IA to accommodate that as part of this redesign too."*

Concrete issues with where these actions live today:

- **Hardcoded inside breadcrumb component.** `project-navigation.tsx` simultaneously owns: breadcrumb assembly, project-level actions, and a details-panel toggle. Three responsibilities, one file.
- **The tooltip text betrays the ambiguity.** *"Redeploy the current active deployment"* — but the user is not necessarily looking at a deployment. They might be on Logs, Env Vars, Settings, anywhere under the project. The action operates on an implicit "current active deployment" that the breadcrumb bar gives no signpost for.
- **Symbols collide.** The same `<DoubleChevronLeft>` icon used for "collapse sidebar" elsewhere is here a "show / hide details panel" toggle. Both live in the chrome, both use the same glyph, neither labels itself in resting state.
- **No concept of a project-level surface** — there's no "Project Overview" page (it was removed: `navigation-configs.tsx:177–183`). So actions that conceptually belong to "the project as a whole" have nowhere natural to live except crammed into the chrome.
- **Per-row deployment actions live elsewhere** (`deployments/components/table/components/actions/...`). The split between "row actions" and "project-wide actions" is fine; the question is whether the project-wide ones should sit in the chrome at all.

### Where could project-level actions go? (open question, not a recommendation)

Three candidate homes that the variations work should test:

- **(a) Stay in the chrome**, but with a clearer slot dedicated to "project-level actions" (vs. "breadcrumb navigation").
- **(b) In an in-page section header**, surfaced on the most-relevant page (e.g., Redeploy on Deployments).
- **(c) On a dedicated project overview page**, the "home" of the project where chrome actions become page actions.

---

## Prior explorations

Dave's existing FigJam: [Deploy: Navbar Redesign](https://www.figma.com/board/bWIYSBO9hPnPcfvBJNy6U4/Deploy--Navbar-Exploration?node-id=2004-168). Contains ~8 screenshot variations and two stickies worth carrying forward:

- *"These two areas work very well imo, and I feel like it might be good to replicate these two areas here in the new navbar."* — capture which areas from the board into the variations work.
- *"I have a feeling at some point, you guys are going to need in-app notifications for stuff. e.g changelog updates, build failures, dependency updates. We don't need it for now, but it'd be good to have one eye on it for the future."* — covered above under Zone 2.

---

## Walkthrough findings

> To populate by clicking through the live dashboard. Capture screenshots + annotations of what feels broken / temporary / inconsistent under each top-level route.

| Route | Notes |
| --- | --- |
| `/[ws]` (workspace root) | _to fill_ |
| `/[ws]/apis` | _to fill_ |
| `/[ws]/apis/[apiId]` | _to fill_ |
| `/[ws]/projects` | _to fill_ |
| `/[ws]/projects/[id]/deployments` | _to fill_ |
| `/[ws]/projects/[id]/logs` | _to fill_ |
| `/[ws]/projects/[id]/env-vars` | _to fill_ |
| `/[ws]/projects/[id]/sentinel-policies` | _to fill_ |
| `/[ws]/projects/[id]/settings` | _to fill_ |
| `/[ws]/ratelimits` | _to fill_ |
| `/[ws]/authorization/roles` | _to fill_ |
| `/[ws]/logs` | _to fill_ |
| `/[ws]/identities` | _to fill_ |
| `/[ws]/audit` | _to fill_ |
| `/[ws]/settings/general` | _to fill_ |

---

## Open questions for the team

1. **Project-level actions** — where do they belong long-term? (a) chrome, (b) page header, (c) project overview page? Resolving this is the load-bearing IA decision for this work.
2. **Product switcher** — does it disappear once Deploy and API Management unify, or does it become permanent? Andreas described it as temporary; what's the long-term picture?
3. **"APIs" as a sidebar label.** Once Deploy ships, "my API" is ambiguous — the HTTP service I deployed, or the set of keys I verify? Andreas's own language is *"API keys UX"* (Apr 1, line 31). Rename candidates: **API Keys** (matches marketing/docs, low risk), **Keyspaces** / **Keyspace management** (matches the data model — `keyAuthId` in `navigation-configs.tsx:105–113` — but introduces jargon and reads recursively under the "API Management" product label), **Authentication** (umbrella that could absorb Authorization + Identities — bigger scope change). Not just a nav fix; affects docs + marketing language, so needs alignment beyond this audit.
4. **Breadcrumbs everywhere, or only in resource scopes?** Today, half the dashboard has none. Is that intentional or accidental?
5. **In-app notifications slot** — yes/no/when? Affects how much room we leave on the right side of the top bar.
6. **Marketing-site brand alignment** — what does the new marketing brand actually look like in nav components? (Out of scope for this audit, but the next sub-issue will need this.)
7. **Mobile** — is the separate `SidebarMobile` component a permanent fixture or should the new navigation be one component that responds to viewport?

---

## Next steps (deferred)

After the audit is reviewed:

1. Pick 2–3 directions from the open questions above.
2. Build a localStorage-backed variant switcher — mirror of `hooks/use-product-selection.ts` (the proven pattern: lazy-init from localStorage, custom `STORAGE_EVENT` for cross-component sync, simple `switch*()` mutator).
3. Stand up each direction as a swappable variant under `components/navigation/variants/`.
4. Mount a dev-only switcher widget (gated on `process.env.NODE_ENV !== "production"`) so the team can A/B them in the same browser session.

These are not part of this audit. They unblock once the team has reacted to this doc.

---

## Source map

- Apr 1 call: `/Users/dave/Developer/unkey/calls/2025-04-01-andreas-thomas.md`
- Apr 7 call: `/Users/dave/Developer/unkey/calls/2025-04-07-oz.md`
- Apr 17 sync: referenced in [DES-29](https://linear.app/unkey/issue/DES-29/vision-dashboard-prototype) (no transcript on disk)
- FigJam: [Deploy: Navbar Redesign](https://www.figma.com/board/bWIYSBO9hPnPcfvBJNy6U4/Deploy--Navbar-Exploration?node-id=2004-168)
