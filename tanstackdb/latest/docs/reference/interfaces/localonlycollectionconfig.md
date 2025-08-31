# LocalOnlyCollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: LocalOnlyCollectionConfig<TExplicit, TSchema, TFallback, TKey>RemarksType ParametersPropertiesgetKey()ParametersitemReturnsid?initialData?onDelete()?ParametersparamsReturnsonInsert()?ParametersparamsReturnsonUpdate()?ParametersparamsReturnsschema?# LocalOnlyCollectionConfig

Copy Markdown[Interface: LocalOnlyCollectionConfig<TExplicit, TSchema, TFallback, TKey>](#interface-localonlycollectionconfigtexplicit-tschema-tfallback-tkey)

Defined in:[packages/db/src/local-only.ts:27](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L27)

Configuration interface for Local-only collection options
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

•**TFallback***extends*Record<string,unknown> =Record<string,unknown>

The fallback type if no explicit or schema type is provided

•**TKey***extends*string|number=string|number

The type of the key returned by getKey
[Properties](#properties)[getKey()](#getkey)ts

```
getKey: (item) => TKey;

```

```
getKey: (item) => TKey;

```

Defined in:[packages/db/src/local-only.ts:38](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L38)
[Parameters](#parameters)[item](#item)

[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>
[Returns](#returns)

TKey
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/local-only.ts:36](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L36)

Standard Collection configuration properties
[initialData?](#initialdata)ts

```
optional initialData: ResolveType<TExplicit, TSchema, TFallback>[];

```

```
optional initialData: ResolveType<TExplicit, TSchema, TFallback>[];

```

Defined in:[packages/db/src/local-only.ts:44](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L44)

Optional initial data to populate the collection with on creation
This data will be applied during the initial sync process
[onDelete()?](#ondelete)ts

```
optional onDelete: (params) => Promise<any>;

```

```
optional onDelete: (params) => Promise<any>;

```

Defined in:[packages/db/src/local-only.ts:77](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L77)

Optional asynchronous handler function called after a delete operation
[Parameters](#parameters-1)[params](#params)

[DeleteMutationFnParams](/db/latest/docs/reference/type-aliases/deletemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>

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

Defined in:[packages/db/src/local-only.ts:51](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L51)

Optional asynchronous handler function called after an insert operation
[Parameters](#parameters-2)[params](#params-1)

[InsertMutationFnParams](/db/latest/docs/reference/type-aliases/insertmutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>

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

Defined in:[packages/db/src/local-only.ts:64](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L64)

Optional asynchronous handler function called after an update operation
[Parameters](#parameters-3)[params](#params-2)

[UpdateMutationFnParams](/db/latest/docs/reference/type-aliases/updatemutationfnparams)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,[LocalOnlyCollectionUtils](/db/latest/docs/reference/interfaces/localonlycollectionutils)>

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

Defined in:[packages/db/src/local-only.ts:37](https://github.com/TanStack/db/blob/main/packages/db/src/local-only.ts#L37)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/localonlycollectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>