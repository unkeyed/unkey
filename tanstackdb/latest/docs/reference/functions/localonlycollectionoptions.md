# localOnlyCollectionOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: localOnlyCollectionOptions()Type ParametersParametersconfigReturnsgcTimegetKey()ParametersitemReturnsid?onDelete()ParametersparamsReturnsonInsert()ParametersparamsReturnsonUpdate()ParametersparamsReturnsschema?startSyncsyncutilsExamples# localOnlyCollectionOptions

Copy Markdown[Function: localOnlyCollectionOptions()](#function-localonlycollectionoptions)ts

```
function localOnlyCollectionOptions<TExplicit, TSchema, TFallback, TKey>(config): object

```

```
function localOnlyCollectionOptions<TExplicit, TSchema, TFallback, TKey>(config): object

```

Defined in:[packages/db/src/local-only.ts:137](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L137)

Creates Local-only collection options for use with a standard Collection

This is an in-memory collection that doesn't sync with external sources but uses a loopback sync config
that immediately "syncs" all optimistic changes to the collection, making them permanent.
Perfect for local-only data that doesn't need persistence or external synchronization.
[Type Parameters](#type-parameters)

•**TExplicit**=unknown

The explicit type of items in the collection (highest priority)

•**TSchema***extends*StandardSchemaV1<unknown,unknown> =never

The schema type for validation and type inference (second priority)

•**TFallback***extends*Record<string,unknown> =Record<string,unknown>

The fallback type if no explicit or schema type is provided

•**TKey***extends*string|number=string|number

The type of the key returned by getKey
[Parameters](#parameters)[config](#config)

[LocalOnlyCollectionConfig](/db/latest/docs/reference/interfaces/localonlycollectionconfig)<TExplicit,TSchema,TFallback,TKey>

Configuration options for the Local-only collection
[Returns](#returns)

object

Collection options with utilities (currently empty but follows the pattern)
[gcTime](#gctime)ts

```
gcTime: number = 0;

```

```
gcTime: number = 0;

```[getKey()](#getkey)ts

```
getKey: (item) => TKey;

```

```
getKey: (item) => TKey;

```[Parameters](#parameters-1)[item](#item)

[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>
[Returns](#returns-1)

TKey
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Standard Collection configuration properties
[onDelete()](#ondelete)ts

```
onDelete: (params) => Promise<any> = wrappedOnDelete;

```

```
onDelete: (params) => Promise<any> = wrappedOnDelete;

```

Wrapper for onDelete handler that also confirms the transaction immediately
[Parameters](#parameters-2)[params](#params)

[DeleteMutationFnParams](/db/latest/docs/reference/type-aliases/deletemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>
[Returns](#returns-2)

Promise<any>
[onInsert()](#oninsert)ts

```
onInsert: (params) => Promise<any> = wrappedOnInsert;

```

```
onInsert: (params) => Promise<any> = wrappedOnInsert;

```

Create wrapper handlers that call user handlers first, then confirm transactions
Wraps the user's onInsert handler to also confirm the transaction immediately
[Parameters](#parameters-3)[params](#params-1)

[InsertMutationFnParams](/db/latest/docs/reference/type-aliases/insertmutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>
[Returns](#returns-3)

Promise<any>
[onUpdate()](#onupdate)ts

```
onUpdate: (params) => Promise<any> = wrappedOnUpdate;

```

```
onUpdate: (params) => Promise<any> = wrappedOnUpdate;

```

Wrapper for onUpdate handler that also confirms the transaction immediately
[Parameters](#parameters-4)[params](#params-2)

[UpdateMutationFnParams](/db/latest/docs/reference/type-aliases/updatemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>
[Returns](#returns-4)

Promise<any>
[schema?](#schema)ts

```
optional schema: TSchema;

```

```
optional schema: TSchema;

```[startSync](#startsync)ts

```
startSync: boolean = true;

```

```
startSync: boolean = true;

```[sync](#sync)ts

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, TKey> = syncResult.sync;

```

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, TKey> = syncResult.sync;

```[utils](#utils)ts

```
utils: LocalOnlyCollectionUtils;

```

```
utils: LocalOnlyCollectionUtils;

```[Examples](#examples)ts

```
// Basic local-only collection
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
  })
)

```

```
// Basic local-only collection
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
  })
)

```ts

```
// Local-only collection with initial data
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
    initialData: [
      { id: 1, name: 'Item 1' },
      { id: 2, name: 'Item 2' },
    ],
  })
)

```

```
// Local-only collection with initial data
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
    initialData: [
      { id: 1, name: 'Item 1' },
      { id: 2, name: 'Item 2' },
    ],
  })
)

```ts

```
// Local-only collection with mutation handlers
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      console.log('Item inserted:', transaction.mutations[0].modified)
      // Custom logic after insert
    },
  })
)

```

```
// Local-only collection with mutation handlers
const collection = createCollection(
  localOnlyCollectionOptions({
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      console.log('Item inserted:', transaction.mutations[0].modified)
      // Custom logic after insert
    },
  })
)

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/localonlycollectionoptions.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>