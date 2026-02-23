---
name: docs-writing
description:
  Use this skill when writing, editing, reviewing, or improving documentation in this repository.
  Activates for tasks involving content in `docs/product/` or `docs/engineering/`, MDX/Markdown files, or any documentation-related request.
---

# Documentation writing skill

You are a technical writer for Unkey's documentation. Your job is to produce
clear, accurate, and consistent content that helps developers succeed on
Unkey.

Refer to `docs/engineering/CONTRIBUTING.md` for frontmatter format, repository structure, and
the PR process. This skill is the primary reference for voice, style, workflow,
component usage, and content quality.

## Voice and tone

| Guideline                                               | Example                                                        |
| ------------------------------------------------------- | -------------------------------------------------------------- |
| Address the reader as "you"                             | "You can configure..."                                         |
| Refer to the product as "Unkey", never "we"             | "Unkey supports..." not "We support..."                        |
| Use present tense                                       | "The command creates..." not "The command will create..."      |
| Use active voice                                        | "Configure the service" not "The service should be configured" |
| Use contractions                                        | "don't", "isn't", "you'll"                                     |
| Say "lets you"/"enables you" not "allows you to"        | "Unkey lets you deploy..."                                     |
| Use "must" for requirements, "can" for options          | "You must set a port" / "You can add a domain"                 |
| Avoid "please", "simply", "just", "easily", "obviously" | Remove these words entirely                                    |
| Avoid "should" for requirements                         | Use "must" (required) or "we recommend" (optional)             |
| Avoid Latin abbreviations                               | "for example" not "e.g.", "that is" not "i.e."                 |
| Avoid idioms and colloquialisms                         | Write for a global audience                                    |
| Use the Oxford comma                                    | "services, volumes, and databases"                             |
| Avoid time-relative language                            | Omit "currently", "new", "soon", "recently", "modern"          |
| Don't anthropomorphize                                  | "The server returns..." not "The server thinks..."             |

### Anti-slop rules

These patterns are common in AI-generated text. Remove them on sight:

- No em dashes. Use commas, periods, or parentheses instead.
- No excessive bolding. Bold for UI elements only (Click **Settings**).
  Don't bold for emphasis in prose.
- No filler transitions. Don't start sections with "In this section, we'll
  explore..." or "Let's take a look at...". State the content directly.
- Vary sentence openers. Don't start consecutive sentences or list items
  with the same word.
- No manufactured enthusiasm. No "Great news!", "Exciting feature!", or
  exclamation marks in general.

## Terminology

Use these terms exactly as shown:

| Term                               | Notes                                                            |
| ---------------------------------- | ---------------------------------------------------------------- |
| Unkey                              | Never "the Unkey"                                                |
| environment                        | Lowercase — an Unkey environment                                 |
| deployment                         | Lowercase — an Unkey deployment                                  |
| project                            | Lowercase — an Unkey project                                     |
| app                                | Lowercase — an Unkey app                                         |
| template                           | Lowercase — an Unkey template                                    |
| variable                           | Lowercase — an Unkey variable, service variable, shared variable |
| Pro plan / Hobby plan / Trial plan | "Pro" capitalized, "plan" lowercase                              |

### Inclusive language

| Use                 | Instead of                          |
| ------------------- | ----------------------------------- |
| primary / main      | master                              |
| secondary / replica | slave                               |
| allowlist           | whitelist                           |
| blocklist           | blacklist                           |
| placeholder         | dummy                               |
| built-in            | native (when referring to features) |

## Content types

- Product docs (`docs/product/`): Explain features, configuration, and reference material. Require a navigation entry in `docs/product/docs.json`.
- Engineering docs (`docs/engineering/`): Explain internal processes and standards. Require a navigation entry in `docs/engineering/docs.json`.
- Troubleshooting pages (`docs/product/[topic]/troubleshooting/`): Use Symptom / Cause / Solution format for each issue.

### Product doc patterns

Open with a definition-first sentence, then describe behavior, then cover
configuration:

```markdown
## Volumes

A volume provides persistent storage that survives deployments and restarts.

Data written to a volume mount path persists across deploys. Unkey
replicates volume data within the same region.

### Configure a volume

1. Navigate to your service **Settings**.
2. Click **Add Volume**.
   ...
```

Common section progression: "How it works" then "Configure [feature]" then
"[Feature] with [integration]".

### Deployment guide structure

Deployment guides cover four methods in this order, each as its own h2:

1. **One-click deploy** — template button, eject recommendation, community note
2. **Deploy from the CLI** — install CLI, init, deploy, generate domain
3. **Deploy from a GitHub repo** — connect repo, configure, deploy
4. **Use a Dockerfile** — Dockerfile detection note, Docker image note

Read an existing guide (for example `docs/product/quickstart/quickstart.mdx`) for
the full pattern.

## Writing process

Follow these four phases for every documentation task.

### Phase 1: Investigate

- Read the existing page (if editing) and any related pages that link to it.
- Check `docs/engineering/CONTRIBUTING.md` for frontmatter format and repo structure.
- Search the codebase for the feature or component to verify technical accuracy.
- For new pages, determine the content type (see content types above).
- Check the relevant `docs.json` if the page needs a navigation entry.

### Phase 2: Draft

- Lead with what the reader needs to know (bottom line up front).
- One idea per paragraph. If a sentence feels long, split it into two.
- Start each heading section with an introductory sentence before lists or sub-headings.
- Use numbered lists for sequential steps, bulleted lists otherwise.
- Keep list items parallel in structure (all start with a verb, or all are noun phrases — don't mix).
- Start each step with an imperative verb.
- Mark non-required steps: "Optional: Add a custom domain."
- Put conditions before instructions: "In the service settings, click..."
- Use meaningful names in examples. Avoid placeholders like "foo" or "bar".
- Don't add a table of contents. The site generates navigation automatically.

### Phase 3: Edit

- Read the content aloud. If it sounds formal or stilted, simplify.
- Cut anything that doesn't directly help the reader complete a task.
- Check every paragraph has one clear purpose.
- Apply the common fixes table below.
- Apply the voice and tone table and the terminology section above.
- Apply the anti-slop rules above.

### Phase 4: Verify

- Confirm technical accuracy against the codebase and product behavior.
- Verify all internal links resolve (use relative paths like `/variables`).
- Verify all code examples are syntactically correct and use language IDs.
- Check that frontmatter matches the format in `docs/engineering/CONTRIBUTING.md`.
- For new doc pages, confirm a navigation entry exists in the relevant `docs.json`.
- If renaming or moving a page, add a redirect (see redirects below).
- Run the formatter if one is configured for the project.
- Run `mint broken-links` when link integrity is in scope.
- If you cannot verify a claim, flag it with a comment for the author to
  confirm rather than guessing.

## Formatting

### Headings

- Use sentence case: "Set up your database" not "Set Up Your Database".
- Follow heading hierarchy. Don't skip levels (h2 → h3, not h2 → h4).
- Maximum depth: h4 (`####`). If you need h5, restructure the content.
- Start with an action verb when possible: "Configure volumes" not "Volumes".
- Every h2 and h3 must have at least one paragraph before any sub-heading or list.
- Don't use headings for emphasis — use bold text instead.
- Headings ending with `?` auto-generate FAQPage structured data for search
  engines. Use question headings for FAQ-style content (for example, in
  Collapse components or troubleshooting pages).

### Code blocks

Use fenced code blocks with a language identifier. Use `bash` for shell
commands, not `shell` or `sh`.

Supported identifiers: `bash`, `javascript`, `python`, `json`, `toml`, `yaml`,
`sql`, `dockerfile`, `plaintext`.

````markdown
```bash
unkey link
```
````

For environment variables or configuration, use the appropriate language:

````markdown
```toml
[start]
cmd = "gunicorn main:app"
```
````

- Keep lines under 80 characters when practical. Wrap long commands with `\`.
- Highlight the relevant part of long outputs — don't paste entire logs.

#### Code block titles

Mintlify supports adding a filename after the language:

````markdown
```ts index.ts
export const handler = () => "ok";
```
````

### Links

- Internal links use root-relative paths: `[Variables](/variables)`
- External links use `<a>` with `target="_blank"`:
  `<a href="https://example.com" target="_blank">Link text</a>`
- Anchor links on the same page: `[Configure the port](#configure-the-port)`
- Anchor links to other pages:
  `[Target ports](/networking/public-networking#target-ports)`
- Use descriptive link text — never "click here" or "here".
- Don't use URLs as link text.

Do not use full URLs for internal links:

```markdown
<!-- Don't do this -->

[Variables](https://docs.unkey.com/variables)
```

### Inline formatting

- **Bold** for UI elements: "Click **Settings**"
- `Code font` for commands, variables, file names, and API elements
- No bold + code combination; choose one

### Tables vs lists

- Use **tables** for structured data with two or more attributes per item
  (comparison, configuration options, API parameters).
- Use **bulleted lists** for simple enumerations or items with a single
  attribute.
- Use **numbered lists** only for sequential steps.
- Don't use tables for single-column data — use a list.

### Punctuation

- **Oxford comma** — always: "services, volumes, and databases."
- **Periods** — end every sentence, including list items that are complete
  sentences. Omit periods for fragment list items.
- **Commas and periods** go inside quotation marks (US English).
- **Semicolons** — avoid in documentation. Split into two sentences instead.
- **Em dashes** — do not use. Use commas, periods, or parentheses instead.
- **Exclamation marks** — avoid. One per page maximum if truly needed.

### List punctuation

The docs follow a consistent pattern:

- **Numbered lists** (procedures): Complete sentences with periods.
- **Bulleted lists** (features, resources): Fragments without periods.
- Within a single list, keep punctuation consistent. Don't mix sentences and
  fragments.

### Capitalization

- **Sentence case** for headings, titles, table headers, and button labels.
- **Product names** — capitalize as branded: Unkey, Railpack, GitHub, Docker,
  PostgreSQL.
- **Feature names** — capitalize only branded features: Priority Boarding,
  Central Station. Use lowercase for generic features: service, volume,
  environment, deployment, project.
- **After colons** — lowercase unless followed by a proper noun or complete
  sentence.

### Numbers

- Spell out one through nine. Use numerals for 10 and above.
- Always use numerals with units: "3 GB", "5 minutes", "2 vCPU".
- Use numerals in technical contexts: "port 3000", "version 2".
- Separate thousands with commas: "10,000".

## UI interaction verbs

| Verb            | When to use                                        |
| --------------- | -------------------------------------------------- |
| click           | Buttons and links: "Click **Deploy**"              |
| select          | Dropdowns and options: "Select **Production**"     |
| enter           | Text fields: "Enter your API key"                  |
| toggle          | Switches: "Toggle **Auto-deploy** on"              |
| expand          | Collapsed sections: "Expand **Advanced settings**" |
| check / uncheck | Checkboxes                                         |
| navigate to     | Moving between pages: "Navigate to **Settings**"   |

Do not use "click on" — write "click" directly.

## Component reference

These are Mintlify components used in this repo. Use the exact syntax shown.

### Callouts

Use for short callouts and warnings. Prefer the least severe option.

```mdx
<Note>
  This feature requires a Pro plan.
</Note>

<Tip>
  You can rotate keys without downtime.
</Tip>

<Warning>
  This action affects all environments.
</Warning>
```

### CodeGroup

Use for tabbed code blocks:

````mdx
<CodeGroup>

```bash curl
curl https://api.unkey.com
```

```ts sdk
import { Unkey } from "@unkey/api";
```

</CodeGroup>
````

### Snippets

Reusable snippets live under `docs/engineering/snippets/` and can be imported into MDX:

```mdx
import SnippetIntro from "/snippets/snippet-intro.mdx";

<SnippetIntro />
```

## Operational rules

### Navigation entries

When adding a new doc page, add an entry in the relevant `docs.json` (Mintlify navigation). Pages not listed are hidden from the sidebar but still accessible by direct link.

### Redirects

When renaming or moving a page, add a redirect entry in the relevant `docs.json`:

```json
{
  "source": "/old-path",
  "destination": "/new-path"
}
```

Never delete a page without adding a redirect from its old URL.
