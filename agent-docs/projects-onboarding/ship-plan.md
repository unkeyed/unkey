---
title: "Projects + onboarding redesign: ship plan"
date: 2026-07-29
last_updated: 2026-07-29
module: dashboard-projects
problem_type: workflow_issue
category: projects-onboarding
tags: [projects, onboarding, navigation, migration, dhonboarding-vibe]
status: living-document
---

# Projects + onboarding redesign: ship plan

Living document. The `dhonboarding-vibe` branch is the agreed map for the next
phase of the Unkey dashboard (signed off in Slack 2026-07-28 by Andreas +
James). This doc tracks how it gets broken into real, tightly-scoped PRs.
Update this file whenever a ticket moves, a decision lands, or the plan
changes — it is the handoff for any future agent or teammate picking this up.

Sequencing v2 (2026-07-29, Dave's re-jig): design-baseline bundle + flag PR
first, then chart colors → empty-state component → project card → projects
list → project overview (last). Nav cutover builds behind the flag in
parallel. Logged-out/auth is a separate final track. Backend runs parallel
(Andreas). Old T-xx ids kept in brackets for continuity with earlier notes.

## Links

- Rendered plan (artifact, reflects v1 sequencing): https://claude.ai/code/artifact/0699c5c3-3d89-4fcf-a470-eaeb38873607
- Slack: #proj-onboarding-projects (C0BGUU04FPG)
- FigJam: https://www.figma.com/board/76nVsbuzG0dRdEF2gL4uYM
- Preview: https://dashboard-git-dhonboarding-vibe-unkey.vercel.app
- Linear project: https://linear.app/unkey/project/everythings-project-221ce186cf5d (empty — tickets not created yet, working locally first)
- Decisions log: ./decisions.md

## Linear tickets (created 2026-07-29, Everythings project)

A-2 ENG-3095 · A-3 ENG-3096 · A-4 ENG-3097 · A-5 ENG-3098 · B-1 ENG-3099 ·
B-2 ENG-3100 · B-3 ENG-3101 · N-1 ENG-3102 · N-2 ENG-3103 (blocked by 3102) ·
N-3 ENG-3104 · N-4 ENG-3105 · N-5 ENG-3107 (blocked by 3095+3104) ·
N-6 ENG-3106 · N-7 ENG-3108 · L-1 ENG-3109 ·
Backend (ALL DB work, one issue to discuss with Andreas) ENG-3093 (due Fri 31 Jul) ·
Comms ENG-3094 (due Tue 5 Aug). Linear is now the tracking source; this doc
stays the context/decision source.

## The shape of the problem

Branch diff vs origin/main: 44 commits, 145 files, +11,277/−1,549. ~60% of
added lines are prototype scaffolding that must never merge. Exactly 5
non-prototype files import prototype code (3 nav files → prototype store;
2 tRPC clients → prototype interception link).

The branch is three kinds of change tangled together:

1. **Visual/UX work** that ships against today's workspace-scoped data.
2. **IA change** (projects-first nav) — shippable now because migration puts
   every legacy workspace into ONE default project, so workspace-scoped
   queries inside a project shell are correct until multi-project exists.
3. **Data-model read path** — columns, write path and backfill ARE shipped on
   main (#6849–#6866, announced done 27 Jul); read path, unique indexes and
   logs are not. URN scoping is in flight (#6782/#6878/#6880). Gates
   multi-project only, not the redesign.

## Track A — design baseline (start now, unflagged)

| # | Ticket | Size | Risk | Status |
|---|--------|------|------|--------|
| A-2 [new] | Flag-only PR: add the projects-nav feature flag (one flag: sidebar + breadcrumbs + landing redirect), default off, plumbing only | XS | low | todo |
| A-3 [T-03] | Chart --chart-* tokens + StatsListCard palette prop (note: APIs list recolors accent→green) | S | low | todo |
| A-4 [new] | Empty-state component family: extract the prototype's dashed-card empty state (AppsEmpty in project-overview.tsx — dashed grayA-4, icon, 13px title, outline actions) into @unkey/ui as hero + in-card variants. REPLACES EmptyHero — migrate its call-sites, delete it. Doc page in web/apps/design (patterns). No flag — pure restyle | M | low | todo |
| A-5 [T-06+07] | PageHeader migrations: ratelimits, then identities (deletes navbar/ControlCloud/refresh — name deletions in PR body; identity detail goes server→client). Anytime, unflagged | M | med | todo |

## Track B — project surfaces (no flag needed — these pages already exist)

Prototype code is the SPEC, not the diff. Data sources all exist
(deploy.project.list, deploy.app.list, api.overview.query + timeseries,
ratelimit batch timeseries). If T-25 lands first, build resource surfaces
straight on project-scoped queries.

| # | Ticket | Size | Risk | Status |
|---|--------|------|------|--------|
| B-1 [T-17] | Project card rebuilt around last deployment + dashed empty state (uses A-4) | M | low | todo |
| B-2 [T-21+22] | Projects list page: grid + quick-access rail (most-used 24h, List+Bars; must handle etherspot scale: 2,153 keyspaces in one default project) + agent setup pill (placement Q3; prompt content is a deliverable). Usage/Compute meter LAST — blocked on billing framing (Q4) [T-23] | L | med | todo |
| B-3 [T-18+19+20] | Project overview page, last, as 3 PRs: (a) shell + apps table on real deploy data; (b) getting-started checklist (GitHub app → repo → deploy → domain, NO keyspace/ratelimit todos) + real-data empty states, one-screen budget on 13"; (c) keyspace/ratelimit resource cards (tabular-nums tooltips, no per-request framing) | L | med | todo |

Cards inventory (the "any other card?" answer): project card (B-1), rail
rows (B-2), keyspace/ratelimit resource cards + deployment rows + checklist
card + agent pill (B-3), no-compute-plan banner (N-2), namespace/keyspace
list cards (restyled via A-5 + A-3). All go through A-4's empty-state family
where they have empty states.

## Track N — nav cutover (behind the A-2 flag)

| # | Ticket | Size | Risk | Status |
|---|--------|------|------|--------|
| N-1 [T-10] | Stripe checkout projectId plumbing (idempotency key change) | XS | low | todo |
| N-2 [T-11] | Create-project full-page form + paywall moves to APP creation (coordinate Flo; hand-lift usePendingSubscribe from prototype page) | M | med | todo |
| N-3 [T-12] | Real last-project hook + resource→project_id lookup (columns exist; replaces prototype findProjectIdForResource) | S | low | todo |
| N-4 [T-13] | Unhide default project (Andreas): display name = workspace name (slug stays 'default'; DB has name=Default — rename backfill or alias, his pick); suppress delete affordance | S | med | todo |
| N-5 [T-14+09] | Nav restructure behind flag: duplicated leaves.ts, sidebar swap, breadcrumbs + project crumb, root keys top-level in the NEW tree (old tree untouched). 2 PRs: crumb machinery, then sidebar. Resolve: UsageBanner, _portalManagementEnabled, use-api-name error swallow | L | HIGH | todo |
| N-6 [T-15] | Project route re-exports + apps page (landing redirect → /apps until B-3a, then /overview) | S | low | todo |
| N-7 [T-16+C-02] | Rollout: comms ticketed as **ENG-3094 (due 2026-08-05)** — outreach + banner + email decision. Internal flag-on is informal (no ticket). Then landing /apis→/, flag on for all, flag + old leaves.ts deletion | M | med | deferred (ENG-3094) |

## Comms track

| # | Ticket | Status |
|---|--------|--------|
| C-01 | Bob migration-scope data | **done 29 Jul** (see decisions.md) |
| C-02 | Top-customer outreach before rollout (~10–15; Andreas has Slack channels with most) | todo — owner unassigned |

## Track L — logged-out / onboarding (last, separate thing)

| # | Ticket | Size | Risk | Status |
|---|--------|------|------|--------|
| L-1 [T-05] | Auth restyle, 2 PRs: sign-in/up, then challenge/MFA (banners go client-only — flag SSR change in PR). Brings its own GitHub/Google brand icons AND the @unkey/ui text-sm form default with it (the cut A-1 baseline bundle used to carry both — the vibe branch deleted auth's local size overrides in favour of text-sm, so they travel together or the overrides come back) | M | low | todo |
| L-2 | /new workspace onboarding alignment | S | low | todo |

## Backend (Andreas-led, parallel)

| # | Ticket | Size | Risk | Status |
|---|--------|------|------|--------|
| T-24 | Unique-index migrations ×3: ratelimit_namespaces(name), identities(external_id) — hot upsert, Fireworks 1.06M rows — roles/permissions slugs. Confirmed NOT covered by in-flight PRs | M | HIGH | todo — needs go signal |
| T-25 | projectProcedure + projectId filters on ~8 list routers | M | med | todo — needs go signal |
| T-26 | Logs project-scoped v1: MySQL ID-set → ClickHouse IN() (CH has no project_id; _v3 tables later). Hold until after cutover | L | HIGH | todo |
| T-27 | URN/RBAC project scoping | L | med | **in flight: #6782 + #6878 + #6880 open; #6853 finalizes migration** |
| T-28 | Unblock second projects + self-serve resource moves | M | med | todo — after T-24/25 |

## Never-ship list (prototype scaffolding on the branch)

- `lib/trpc/prototype-handlers.ts` + `prototype-link.ts` + registrations in
  `react-query-provider.tsx` and `lib/collections/client.ts`. The
  `workspace.getCurrent → fakeWorkspace()` fallback would mask auth bugs.
- Dev backdoor in `lib/trpc/routers/deploy/project/create.ts` (direct MySQL
  insert when NODE_ENV=development, bypasses ctrl).
- All of `projects/_components/prototype/` (11 files).
- Dead overview options: `option-hero/stats/hub/hybrid*` + `overview-mocks.ts`
  + `agent-prompt.tsx` (8 files, superseded by `project-overview.tsx`).
- Overview prototype shell (`overview-prototype.tsx`, `overview-data.ts`,
  debug command). `project-overview.tsx` is the SPEC for B-3.
- `onboarding-mocks/` at repo root.
- `use-api-name.ts` error swallowing; commented-out UsageBanner.
- Dave's stale PRs to tidy: #6749 (old prototype PR — close in favour of the
  breakout), #6876 (nav banner removal — review/land or close).

## Open questions (gate specific tickets)

1. ~~Feature flag or hard cutover?~~ **DECIDED 29 Jul: ONE flag (sidebar + breadcrumbs + landing), duplicated leaves.ts.** Remaining: internal-first rollout order, removal criteria
2. ~~Default project presentation~~ **DECIDED 29 Jul: displays the workspace name** (slug stays 'default'; rename vs alias = Andreas's pick)
3. ~~Agent prompt placement~~ **DECIDED 29 Jul: as vibed — projects list rail + project overview. Navbar idea shelved for now**
4. Usage/billing framing → **path decided 29 Jul: vibe the proper meter on dhonboarding-vibe FIRST (immediate task, brief in decisions.md), get Andreas/James sign-off, then ticket it.** B-2 ships without the meter until then
5. ~~Apps page fate~~ **DECIDED 29 Jul: survives — overview shows the list with "view all" → dedicated Apps page**
6. Authorization-in-project redirect compromise: ship it behind the flag, revisit if complaints
7. Andreas go signal for T-24 + T-25 → **ticketed: ENG-3093, due Fri 2026-07-31, assigned Dave.** Verified 29 Jul that indexes + read path are genuinely untouched (schema on origin/main, #6853 diff, PR search)
8. ~~Email at rollout?~~ **Deferred into ENG-3094 (due 2026-08-05) along with outreach ownership + banner copy**
