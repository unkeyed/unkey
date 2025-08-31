# Installation | TanStack DB Docs

TanStackDB v0AutoFrameworkReactVersionLatest Search... + KMenuHomeFrameworksContributorsGitHub Discord Getting StartedOverviewcoreQuick StartcoreInstallationcoreReact AdapterreactGuidesLive QueriescoreError HandlingcoreCreating Collection Options CreatorscoreCollectionsQuery CollectioncoreElectric CollectioncoreAPI ReferenceCore API ReferencecoreCollectioncorecreateCollectioncoreliveQueryCollectionOptionscorecreateLiveQueryCollectioncorecreateOptimisticActioncorecreateTransactioncoreElectric DB CollectioncoreelectricCollectionOptionscoreQuery DB CollectioncorequeryCollectionOptionscoreReact HooksreactuseLiveQueryreact[TanStack](/)[DB v0](/db)AutoSearch... + KFrameworkReactVersionLatestMenu

- [Home](/db/latest)
- [Frameworks](/db/latest/docs/framework)
- [Contributors](/db/latest/docs/contributors)
- [GitHub](https://github.com/tanstack/db)
- [Discord](https://tlinz.com/discord)Getting Started

- [Overviewcore](/db/latest/docs/overview)
- [Quick Startcore](/db/latest/docs/quick-start)
- [Installationcore](/db/latest/docs/installation)
- [React Adapterreact](/db/latest/docs/framework/react/adapter)Guides

- [Live Queriescore](/db/latest/docs/guides/live-queries)
- [Error Handlingcore](/db/latest/docs/guides/error-handling)
- [Creating Collection Options Creatorscore](/db/latest/docs/guides/collection-options-creator)Collections

- [Query Collectioncore](/db/latest/docs/collections/query-collection)
- [Electric Collectioncore](/db/latest/docs/collections/electric-collection)API Reference

- [Core API Referencecore](/db/latest/docs/reference/index)
- [Collectioncore](/db/latest/docs/reference/interfaces/collection)
- [createCollectioncore](/db/latest/docs/reference/functions/createcollection)
- [liveQueryCollectionOptionscore](/db/latest/docs/reference/functions/livequerycollectionoptions)
- [createLiveQueryCollectioncore](/db/latest/docs/reference/functions/createlivequerycollection)
- [createOptimisticActioncore](/db/latest/docs/reference/functions/createoptimisticaction)
- [createTransactioncore](/db/latest/docs/reference/functions/createtransaction)
- [Electric DB Collectioncore](/db/latest/docs/reference/electric-db-collection/index)
- [electricCollectionOptionscore](/db/latest/docs/reference/electric-db-collection/functions/electriccollectionoptions)
- [Query DB Collectioncore](/db/latest/docs/reference/query-db-collection/index)
- [queryCollectionOptionscore](/db/latest/docs/reference/query-db-collection/functions/querycollectionoptions)
- [React Hooksreact](/db/latest/docs/framework/react/reference/index)
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageReactSolidSvelteVueVanilla JSCollection PackagesQuery CollectionLocal CollectionsSync EnginesElectric CollectionTrailBase Collection# Installation

Copy MarkdownEach supported framework comes with its own package. Each framework package re-exports everything from the core@tanstack/dbpackage.
[React](#react)sh

```
npm install @tanstack/react-db

```

```
npm install @tanstack/react-db

```

TanStack DB is compatible with React v16.8+
[Solid](#solid)sh

```
npm install @tanstack/solid-db

```

```
npm install @tanstack/solid-db

```[Svelte](#svelte)sh

```
npm install @tanstack/svelte-db

```

```
npm install @tanstack/svelte-db

```[Vue](#vue)sh

```
npm install @tanstack/vue-db

```

```
npm install @tanstack/vue-db

```

TanStack DB is compatible with Vue v3.3.0+
[Vanilla JS](#vanilla-js)sh

```
npm install @tanstack/db

```

```
npm install @tanstack/db

```

Install the the core@tanstack/dbpackage to use DB without a framework.
[Collection Packages](#collection-packages)

TanStack DB also provides specialized collection packages for different data sources and storage needs:
[Query Collection](#query-collection)

For loading data using TanStack Query:
sh

```
npm install @tanstack/query-db-collection

```

```
npm install @tanstack/query-db-collection

```

UsequeryCollectionOptionsto fetch data into collections using TanStack Query. This is perfect for REST APIs and existing TanStack Query setups.
[Local Collections](#local-collections)

Local storage and in-memory collections are included with the framework packages:

- **LocalStorageCollection**- For persistent local data that syncs across browser tabs
- **LocalOnlyCollection**- For temporary in-memory data and UI state

Both uselocalStorageCollectionOptionsandlocalOnlyCollectionOptionsrespectively, available from your framework package (e.g.,@tanstack/react-db).
[Sync Engines](#sync-engines)[Electric Collection](#electric-collection)

For real-time sync with[ElectricSQL](https://electric-sql.com):
sh

```
npm install @tanstack/electric-db-collection

```

```
npm install @tanstack/electric-db-collection

```

UseelectricCollectionOptionsto sync data from Postgres databases through ElectricSQL shapes. Ideal for real-time, local-first applications.
[TrailBase Collection](#trailbase-collection)

For syncing with[TrailBase](https://trailbase.io)backends:
sh

```
npm install @tanstack/trailbase-db-collection

```

```
npm install @tanstack/trailbase-db-collection

```

UsetrailBaseCollectionOptionsto sync records from TrailBase's Record APIs with built-in subscription support.[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/installation.md)[Quick Start](/db/latest/docs/quick-start)[React Adapter](/db/latest/docs/framework/react/adapter)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>