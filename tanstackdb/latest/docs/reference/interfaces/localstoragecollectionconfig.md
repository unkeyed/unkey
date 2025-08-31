# LocalStorageCollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: LocalStorageCollectionConfig<TExplicit, TSchema, TFallback>RemarksType ParametersPropertiesgetKey()ParametersitemReturnsid?onDelete()?ParametersparamsReturnsonInsert()?ParametersparamsReturnsonUpdate()?ParametersparamsReturnsschema?storage?storageEventApi?storageKeysync?# LocalStorageCollectionConfig

Copy Markdown[Interface: LocalStorageCollectionConfig<TExplicit, TSchema, TFallback>](#interface-localstoragecollectionconfigtexplicit-tschema-tfallback)

Defined in:[packages/db/src/local-storage.ts:61](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L61)

Configuration interface for localStorage collection options
[Remarks](#remarks)

Type resolution follows a priority order:

1. If you provide an explicit type via generic parameter, it will be used
2. If no explicit type is provided but a schema is, the schema's output type will be inferred
3. If neither explicit type nor schema is provided, the fallback type will be used

You should provide EITHER an explicit type OR a schema, but not both, as they would conflict.
[Type Parameters](#type-parameters)

•**TExplicit**=unknown

The explicit type of items in the collection (highest priority)

•**TSchema***extends*StandardSchemaV1=never

The schema type for validation and type inference (second priority)

•**TFallback***extends*object=Record<string,unknown>

The fallback type if no explicit or schema type is provided
[Properties](#properties)[getKey()](#getkey)ts

```
getKey: (item) => string | number;

```

```
getKey: (item) => string | number;

```

Defined in:[packages/db/src/local-storage.ts:88](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L88)
[Parameters](#parameters)[item](#item)

[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)
[Returns](#returns)

string|number
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/local-storage.ts:86](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L86)

Collection identifier (defaults to "local-collection:{storageKey}" if not provided)
[onDelete()?](#ondelete)ts

```
optional onDelete: (params) => Promise<any>;

```

```
optional onDelete: (params) => Promise<any>;

```

Defined in:[packages/db/src/local-storage.ts:114](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L114)

Optional asynchronous handler function called before a delete operation
[Parameters](#parameters-1)[params](#params)

[DeleteMutationFnParams](/db/latest/docs/reference/type-aliases/deletemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-1)

Promise<any>

Promise resolving to any value
[onInsert()?](#oninsert)ts

```
optional onInsert: (params) => Promise<any>;

```

```
optional onInsert: (params) => Promise<any>;

```

Defined in:[packages/db/src/local-storage.ts:96](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L96)

Optional asynchronous handler function called before an insert operation
[Parameters](#parameters-2)[params](#params-1)

[InsertMutationFnParams](/db/latest/docs/reference/type-aliases/insertmutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-2)

Promise<any>

Promise resolving to any value
[onUpdate()?](#onupdate)ts

```
optional onUpdate: (params) => Promise<any>;

```

```
optional onUpdate: (params) => Promise<any>;

```

Defined in:[packages/db/src/local-storage.ts:105](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L105)

Optional asynchronous handler function called before an update operation
[Parameters](#parameters-3)[params](#params-2)

[UpdateMutationFnParams](/db/latest/docs/reference/type-aliases/updatemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-3)

Promise<any>

Promise resolving to any value
[schema?](#schema)ts

```
optional schema: TSchema;

```

```
optional schema: TSchema;

```

Defined in:[packages/db/src/local-storage.ts:87](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L87)
[storage?](#storage)ts

```
optional storage: StorageApi;

```

```
optional storage: StorageApi;

```

Defined in:[packages/db/src/local-storage.ts:75](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L75)

Storage API to use (defaults to window.localStorage)
Can be any object that implements the Storage interface (e.g., sessionStorage)
[storageEventApi?](#storageeventapi)ts

```
optional storageEventApi: StorageEventApi;

```

```
optional storageEventApi: StorageEventApi;

```

Defined in:[packages/db/src/local-storage.ts:81](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L81)

Storage event API to use for cross-tab synchronization (defaults to window)
Can be any object that implements addEventListener/removeEventListener for storage events
[storageKey](#storagekey)ts

```
storageKey: string;

```

```
storageKey: string;

```

Defined in:[packages/db/src/local-storage.ts:69](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L69)

The key to use for storing the collection data in localStorage/sessionStorage
[sync?](#sync)ts

```
optional sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```

```
optional sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```

Defined in:[packages/db/src/local-storage.ts:89](https://github.com/TanStack/db/blob/main/packages/db/src/local-storage.ts#L89)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/localstoragecollectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>