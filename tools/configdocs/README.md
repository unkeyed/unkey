# Configuration references for Notion

`configdocs` renders a Go config struct as Notion-flavored Markdown using
`config.md.tmpl`. It reads field names, Go types, TOML/config tags, and comments.
Operators maintain descriptions in the struct comments, not a separate Markdown
file. The tool does not infer runtime behavior or generate operational advice.

Headings follow struct nesting from H1 through H4. Deeper fields stay at H4,
Notion's Markdown limit. Every heading keeps its full dotted field path.

Go doc references such as `[config.Load]`, `[Config.Validate]`, and `[Config.Port]`
link to their declarations in GitHub at `--ref`. Package references link to their
repository directories. Import aliases and comments inherited from nested types
use the declaring file's context. Standard-library and third-party references
link to `pkg.go.dev`. Unresolved or ambiguous symbol references stay as text;
code examples and explicit links are preserved.

## Preview

Run from the repository root:

```bash
mise exec -- go run ./tools/configdocs \
  --file svc/frontline/config.go --struct Config
```

`--file` is required. `--struct` defaults to `Config` and must name a struct
declared in that file. Sibling files and imported repository packages are read
to resolve nested types and comment references.
Without `--notion-page-id`, the tool prints Markdown and makes no HTTP requests.

The extractor follows named structs, pointers, and slices in the service package
and other packages in this repository. It includes exported fields with TOML
names and skips `toml:"-"`. Imported third-party types, interfaces, and maps
remain leaf fields. Custom TOML unmarshaling and validation methods are not
interpreted. Recursive config types and ambiguous type declarations fail generation.
Parsing is source-based, not a build: platform build tags are not evaluated.

## Upload

Replace `<page-id>` with an existing page ID and make `NOTION_TOKEN` available
through the environment. Share the destination page with the Notion integration
and give the integration update-content access.

```bash
mise exec -- go run ./tools/configdocs \
  --file svc/frontline/config.go --struct Config \
  --notion-page-id="<page-id>"
```

The tool renders and uploads in one invocation. It replaces the page body with
the generated reference. Human edits to that body are overwritten. Keep human
documentation on a separate page. It does not create pages or change their
titles, parents, properties, verification, or permissions. Notion rejects a
replacement that would delete child pages or databases.

After uploading, the tool locks the page against accidental edits in the Notion
app. The lock does not block API updates, so later uploads need no unlock step.
If locking fails, the tool reports that the content was uploaded but the page
lock failed and exits nonzero.

Errors exit nonzero, including missing credentials, invalid page IDs, and Notion
failures. Uploads use a one-minute HTTP timeout and are not retried automatically.

## CI

The [configuration sync workflow](../../.github/workflows/sync-configdocs.yaml)
runs on every push to `main` and supports manual dispatch. It serializes runs,
checks out the latest `main`, and passes that checkout's commit to `--ref`.
Queued runs and reruns publish from `main`, not an older triggering commit.

The workflow maps each service's config file to a `Configuration` child page
inside its Notion service page. It covers API, Control API, Control Worker,
Frontline, Heimdall, Krane, Logdrain, and Vault. Kitchensink has no config struct
and is not included. To add a service, create its child page, share it with the
Notion integration, and add the config file and page ID to the workflow's list.

Set the repository's `NOTION_TOKEN` Actions secret to an integration token with
update-content access to all destination pages. The workflow uploads services
sequentially, attempts the remaining services after a failure, and fails the job
if any sync fails. Each successful sync replaces and locks its destination page.

Every run publishes all eight references, including changes to shared config
types and the template. No generated Markdown file or front matter is needed.

Run the tool's tests with `mise exec -- rask ./tools/configdocs`.
