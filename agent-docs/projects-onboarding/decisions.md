---
title: "Projects + onboarding redesign: decisions log"
date: 2026-07-29
last_updated: 2026-07-29
module: dashboard-projects
problem_type: workflow_issue
category: projects-onboarding
tags: [projects, onboarding, decisions, migration]
status: living-document
---

# Decisions log — projects + onboarding redesign

Append-only. Newest at the top. Each entry: date, decision, source.

## 2026-07-29 — baseline bundle (A-1) cut entirely (Dave)

- The bundle PR (card border → grayA-4, form 13px → text-sm, brand
  GitHub/Google icons, list search input restyle) is dropped — "not
  important". Track A now starts at A-2 (the flag).
- Knock-on: L-1 (auth restyle) absorbs the icons + the @unkey/ui text-sm
  default when it happens (auth deleted its local size overrides expecting
  text-sm). Card border + search-input restyles die unless resurrected
  inside some later surface PR.

## 2026-07-29 — rollout tail deferred + flag approach simplified (Dave)

- Rollout comms (outreach + banner + email decision) deferred: ticketed as
  ENG-3094, assigned Dave, due 2026-08-05 (Linear due date = the reminder).
- Flag approach: the flag lands as a really early ticket (A-2) and
  EVERYTHING on top is additive behind it. Flagging internal users in
  happens informally soon — no ticket needed. No formal removal criteria
  set; old-tree deletion stays in N-7's scope.

## 2026-07-29 — dedicated Apps page survives (Dave)

- The standalone Apps page stays alongside the overview: overview shows the
  apps list with a "view all" button pointing at it. Jul 27's "possibly fold
  it into overview" thought is closed. N-6 keeps the apps page; B-3a's
  overview table links to it.

## 2026-07-29 — billing/usage meter: vibe it properly first (Dave)

- The rail's Usage/Compute meter does NOT get ticketed yet. Immediate next
  vibe task on dhonboarding-vibe: design the proper billing/usage meter on
  the branch, put it in front of Andreas/James, THEN ticket it.
- Brief for that session: Andreas rejects anything reading as per-request
  pricing (and floated "$ amount only… but that defeats the purpose");
  James rejects "requests" terminology for API-management usage; existing
  building blocks are UsageMeter/CircleProgress + trpc.billing.queryUsage +
  workspace quotas; Compute spend-vs-included-credits gated by
  hasComputePlan; scenarios new/migrated/active.
- Dave 2c (29 Jul): terminology and framing must MATCH THE ACTUAL BILLING
  PAGE — the card is a mini of what billing/settings already says, not a
  new vocabulary. Also: keep metrics and billing separate (standing
  Flo/Andreas constraint); the card answers a billing question, activity
  lives in the charts.
- Explored 4 variants (Statement/Invoice/Ledger/Pulse) via a picker
  harness. **Dave picked LEDGER (29 Jul)**: two labelled product sections —
  "API management" ("Verifications & ratelimits", formatNumber pair + thin
  meter) and "Compute" ("Usage this period", "$X of $Y credits" or "No
  active plan / Choose a plan →") — days-left footer, Billing link.
  Promoted into rail.tsx UsageCard for all 3 scenarios; harness deleted.
  Fill turns warning-9 at 80%, error-9 at 100%. Next: put it in front of
  Andreas/James, then this becomes the spec for B-2's meter ticket.

## 2026-07-29 — agent prompt placement stays as vibed (Dave)

- Two placements, per the prototype: projects LIST (rail pill) and project
  OVERVIEW (empty-state / checklist context). Andreas's omnipresent-navbar
  idea is ignored for now — revisit post-launch if the "always need the
  snippet" argument proves out. Nothing folds into N-5.

## 2026-07-29 — paywall move confirmed (Dave)

- Intended v1 behavior: project creation is FREE; the Compute gate +
  pendingSubscribe → Stripe → return flow moves to APP creation (first
  deploy). Goes through Flo review as part of N-2.

## 2026-07-29 — go-signal ticketed, not sent yet (Dave)

- ENG-3093 (Everythings project, assigned Dave, due Fri 2026-07-31):
  send Andreas the go on unique-index migrations + read path. Linear
  due-date notification is the reminder. Verified 29 Jul the indexes and
  read path are genuinely untouched on origin/main.

## 2026-07-29 — sequencing v2 + empty-state componentization (Dave)

- PR groupings re-jigged (ship-plan.md rewritten to tracks A/B/N/L +
  backend): ONE baseline bundle PR (card border + form text-sm + brand
  icons + search input), then a flag-only PR immediately, then chart
  colors → empty-state component → project card → projects list →
  project overview LAST. Nav cutover builds behind the flag in parallel;
  destination pages ship first, flag flip reveals a finished world.
- Logged-out/auth restyle deferred to a separate final track.
- The prototype's dashed-card empty state (AppsEmpty in
  project-overview.tsx) becomes a shared component in @unkey/ui and
  **REPLACES EmptyHero** (Dave, later same day) — new family with hero +
  in-card variants, EmptyHero call-sites migrated then deleted. Doc page
  in web/apps/design, adopt everywhere. No feature flag needed for
  empty-state restyles.

## 2026-07-29 — backend state correction: Andreas is ahead of the plan

Source: Andreas's Jul 27 announcement in #dev (p1785166918803969) + PR audit.

- Migration officially DONE and announced: apis, keyspaces, permissions,
  roles, ratelimit_namespaces backfilled into a `slug=default name=Default`
  project per workspace; every create endpoint upserts it. #6853 (open)
  drops the temporary `DEFAULT ''` and deletes the migration script —
  i.e. backfill is verified complete.
- **T-27 (URN extension) is already IN FLIGHT, not a later phase:** #6782
  scopes keyspace/key/ratelimit/RBAC URNs under projects (legacy tuple
  fallbacks kept, so no customer-visible authz change yet); #6878 (URN
  parser: hierarchical wildcard paths) + #6880 (RBAC scope containment,
  stacked on 6878) are open. Andreas said this is his next focus, and it
  unblocks James adding WorkOS OAuth to the CLI — the agent-handoff auth
  piece (relates to T-22).
- #6782 explicitly does NOT change uniqueness constraints, list behavior,
  DB predicates, or public APIs → **T-24 (unique indexes) and T-25 (read
  path) remain genuinely un-started.** The go-signal to Andreas is now just
  those two.
- T-13 wrinkles: Andreas filters the default project out of the projects
  page deliberately ("I don't want people to delete it" — deletion wouldn't
  break anything but would need manual repair). Unhiding must suppress the
  delete affordance. And the backfill wrote `name=Default`, so Dave's
  workspace-name decision needs either a rename backfill or display-time
  aliasing — Andreas to pick.

## 2026-07-29 — nav cutover ships behind a feature flag (Dave)

- The projects-first nav (T-14/T-15/T-16) goes behind ONE feature flag
  covering sidebar + breadcrumbs + landing redirect — a workspace flips
  atomically, no half-states.
- `leaves.ts` gets duplicated for the transition rather than threading
  conditionals through one tree; the old copy is deleted when the flag is
  removed.
- Still open: rollout order (recommend internal workspaces first — Flo is
  dogfooding heavily) and flag-removal criteria.

## 2026-07-29 — default project is named after the workspace (Dave)

- The migrated default project takes the workspace's name, not "Default".
- Note for T-13/backend: the projects table has UNIQUE(workspace_id, slug)
  and the existing default-project machinery matches on BINARY slug =
  'default' (EnsureDefaultProject, FindDefaultProjectByWorkspaceID). Keep
  slug 'default' as the internal marker and treat the display NAME as the
  workspace name — renaming the slug would break the lazy-create/lookup
  path. Confirm approach with Andreas.

## 2026-07-29 — migration scope data (Bob + Andreas, Slack)

- 406 active workspaces (30d traffic) will see the change; ~380 fine for
  silent auto-migration; ~20+ have complex structure.
- Outreach list ~10–15: top-traffic set + etherspot + taoshi (keyspace
  sprawl), steamsets (83M verifications) next, Fireworks, Daymoon.
- **Correction (Andreas): Daymoon are NOT the keyspace-per-customer case.**
- etherspot = 2,153 keyspaces in one workspace → their default project sets
  the scale bar for the rail/overview/most-used UI (T-21).
- Fireworks identities confirmed 1,060,406; next largest is 16k → the
  identities unique-index migration (T-24) is effectively a one-customer risk.
- Andreas: shared Slack channels exist with most of the top 10.
- Follow-up idea (Andreas): WorkOS user-activity may be a better "who
  notices" signal than API traffic.

## 2026-07-28 — direction signed off (Slack thread on preview link)

- Andreas: "I like this a lot". James: "I do really like everything".
- Nits to fold into build tickets, not blockers: billing card must not read
  as per-request pricing (Andreas); "requests" is wrong terminology for API
  management usage (James); tooltip number jitter → tabular-nums + pad to max
  digits; Documentation CTA should deep-link contextually, not to docs
  landing; one-page-no-scroll concern on overview (13" budget).

## 2026-07-28 — overview layout locked (Slack)

- Single-column ("option 2") project overview. Andreas + Dave agreed.
- Getting-started checklist: install GitHub app → connect repo → deploy →
  add custom domain. NO keyspace/ratelimit todos (Andreas: orthogonal to
  "deployed app asap"). Agent step maybe an omnipresent navbar button
  instead of a checklist item — undecided.

## 2026-07-27 — canvas/graph overview rejected (Granola)

- No meaningful relations between apps at project level; unconnected cards
  are a worse list. Simple overview with apps primary, connectors secondary.
- "Bindings" naming questioned — "connections" more accurate.
- Possibly remove the dedicated Apps page (fold into overview) — open.

## 2026-07-13/14 — migration + onboarding model (Andreas call, Granola)

- Migrate ALL legacy resources into ONE default project per workspace (not
  per-keyspace — breaks identities). Nobody lands empty.
- Project-scoped: keyspaces, apps, identities (projectId + per-project
  external_id dedupe), ratelimits, roles/permissions, logs. Workspace-level:
  audit logs, billing/org settings, root keys.
- Comms: in-app banner at rollout + personal outreach to top 10–20 BEFORE
  rollout (advance notice, not permission). Check Bob first (done 29 Jul).
  Whether a mass email accompanies rollout is unresolved (notes conflict).
- Agent-first onboarding is the core concept: web UI vs agent handoff via
  copyable markdown prompt; auth popup Wrangler-style. The prompt content is
  most of the work. Caveat: Unkey not in model training data yet.
- Ratelimits + API-key setup OUT of main onboarding (dedicated /ratelimit
  path); main flow defaults to deploy. Deploy requires paid plan.
- Nav rules (later calls): breadcrumbs capped at 3 levels; logs stay at
  project level, one view, filters via query params; request logs → opt-in
  Sentinel policy, not a default nav item.

## Backend readiness (audited 2026-07-29 against origin/main)

- DONE on main: project_id columns + indexes on apis, key_auth,
  ratelimit_namespaces, identities, roles, permissions; all write paths tag
  the default project; EnsureDefaultProject (Go + TS mirror); backfill tool
  `web/tools/migrate/project-ownership.ts` (#6849/#6850/#6851/#6866).
- MISSING: entire read path (zero non-deploy queries filter project_id);
  projectProcedure; deploy.project.list hides slug='default'; 3
  workspace-scoped unique indexes block two-project coexistence; ClickHouse
  API-management tables have no project_id; URNs project-scoped for
  identities only (#6732) and enforced with Project("*") wildcards.
