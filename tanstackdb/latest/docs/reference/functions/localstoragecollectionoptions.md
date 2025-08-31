# localStorageCollectionOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: localStorageCollectionOptions()Type ParametersParametersconfigReturnsgetKey()ParametersitemReturnsidonDelete()ParametersparamsReturnsonInsert()ParametersparamsReturnsonUpdate()ParametersparamsReturnsschema?syncType declarationmanualTrigger()?Returnsutilsutils.clearStorageutils.getStorageSizeExamples# localStorageCollectionOptions

Copy Markdown[Function: localStorageCollectionOptions()](#function-localstoragecollectionoptions)ts

```
function localStorageCollectionOptions<TExplicit, TSchema, TFallback>(config): object

```

```
function localStorageCollectionOptions<TExplicit, TSchema, TFallback>(config): object

```

Defined in:[packages/db/src/local-storage.ts:205](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L205)

Creates localStorage collection options for use with a standard Collection

This function creates a collection that persists data to localStorage/sessionStorage
and synchronizes changes across browser tabs using storage events.
[Type Parameters](#type-parameters)

•**TExplicit**=unknown

The explicit type of items in the collection (highest priority)

•**TSchema***extends*StandardSchemaV1<unknown,unknown> =never

The schema type for validation and type inference (second priority)

•**TFallback***extends*object=Record<string,unknown>

The fallback type if no explicit or schema type is provided
[Parameters](#parameters)[config](#config)

[LocalStorageCollectionConfig](/db/latest/docs/reference/interfaces/localstoragecollectionconfig)<TExplicit,TSchema,TFallback>

Configuration options for the localStorage collection
[Returns](#returns)

object

Collection options with utilities including clearStorage and getStorageSize
[getKey()](#getkey)ts

```
getKey: (item) => string | number;

```

```
getKey: (item) => string | number;

```[Parameters](#parameters-1)[item](#item)

[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)
[Returns](#returns-1)

string|number
[id](#id)ts

```
id: string = collectionId;

```

```
id: string = collectionId;

```[onDelete()](#ondelete)ts

```
onDelete: (params) => Promise<any> = wrappedOnDelete;

```

```
onDelete: (params) => Promise<any> = wrappedOnDelete;

```[Parameters](#parameters-2)[params](#params)

[DeleteMutationFnParams](/db/latest/docs/reference/type-aliases/deletemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>
[Returns](#returns-2)

Promise<any>
[onInsert()](#oninsert)ts

```
onInsert: (params) => Promise<any> = wrappedOnInsert;

```

```
onInsert: (params) => Promise<any> = wrappedOnInsert;

```[Parameters](#parameters-3)[params](#params-1)

[InsertMutationFnParams](/db/latest/docs/reference/type-aliases/insertmutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>
[Returns](#returns-3)

Promise<any>
[onUpdate()](#onupdate)ts

```
onUpdate: (params) => Promise<any> = wrappedOnUpdate;

```

```
onUpdate: (params) => Promise<any> = wrappedOnUpdate;

```[Parameters](#parameters-4)[params](#params-2)

[UpdateMutationFnParams](/db/latest/docs/reference/type-aliases/updatemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>
[Returns](#returns-4)

Promise<any>
[schema?](#schema)ts

```
optional schema: TSchema;

```

```
optional schema: TSchema;

```[sync](#sync)ts

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number> & object;

```

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number> & object;

```[Type declaration](#type-declaration)[manualTrigger()?](#manualtrigger)ts

```
optional manualTrigger: () => void;

```

```
optional manualTrigger: () => void;

```[Returns](#returns-5)

void
[utils](#utils)ts

```
utils: object;

```

```
utils: object;

```[utils.clearStorage](#utilsclearstorage)ts

```
clearStorage: ClearStorageFn;

```

```
clearStorage: ClearStorageFn;

```[utils.getStorageSize](#utilsgetstoragesize)ts

```
getStorageSize: GetStorageSizeFn;

```

```
getStorageSize: GetStorageSizeFn;

```[Examples](#examples)ts

```
// Basic localStorage collection
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    getKey: (item) => item.id,
  })
)

```

```
// Basic localStorage collection
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    getKey: (item) => item.id,
  })
)

```ts

```
// localStorage collection with custom storage
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    storage: window.sessionStorage, // Use sessionStorage instead
    getKey: (item) => item.id,
  })
)

```

```
// localStorage collection with custom storage
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    storage: window.sessionStorage, // Use sessionStorage instead
    getKey: (item) => item.id,
  })
)

```ts

```
// localStorage collection with mutation handlers
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      console.log('Item inserted:', transaction.mutations[0].modified)
    },
  })
)

```

```
// localStorage collection with mutation handlers
const collection = createCollection(
  localStorageCollectionOptions({
    storageKey: 'todos',
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      console.log('Item inserted:', transaction.mutations[0].modified)
    },
  })
)

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/localstoragecollectionoptions.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>