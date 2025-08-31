# electricCollectionOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: electricCollectionOptions()Type ParametersParametersconfigReturnsgetKey()ParametersitemReturnsid?onDeleteonInsertonUpdateschema?syncutilsutils.awaitTxId# electricCollectionOptions

Copy Markdown[Function: electricCollectionOptions()](#function-electriccollectionoptions)ts

```
function electricCollectionOptions<TExplicit, TSchema, TFallback>(config): object

```

```
function electricCollectionOptions<TExplicit, TSchema, TFallback>(config): object

```

Defined in:[packages/electric-db-collection/src/electric.ts:285](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L285)

Creates Electric collection options for use with a standard Collection
[Type Parameters](#type-parameters)

•**TExplicit***extends*Row<unknown> =Row<unknown>

The explicit type of items in the collection (highest priority)

•**TSchema***extends*StandardSchemaV1<unknown,unknown> =never

The schema type for validation and type inference (second priority)

•**TFallback***extends*Row<unknown> =Row<unknown>

The fallback type if no explicit or schema type is provided
[Parameters](#parameters)[config](#config)

[ElectricCollectionConfig](/db/latest/docs/reference/electric-db-collection/interfaces/electriccollectionconfig)<TExplicit,TSchema,TFallback>

Configuration options for the Electric collection
[Returns](#returns)

object

Collection options with utilities
[getKey()](#getkey)ts

```
getKey: (item) => string | number;

```

```
getKey: (item) => string | number;

```[Parameters](#parameters-1)[item](#item)

ResolveType
[Returns](#returns-1)

string|number
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

All standard Collection configuration properties
[onDelete](#ondelete)ts

```
onDelete: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnDelete;

```

```
onDelete: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnDelete;

```[onInsert](#oninsert)ts

```
onInsert: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnInsert;

```

```
onInsert: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnInsert;

```[onUpdate](#onupdate)ts

```
onUpdate: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnUpdate;

```

```
onUpdate: 
  | undefined
  | (params) => Promise<{
  txid: number | number[];
 }> = wrappedOnUpdate;

```[schema?](#schema)ts

```
optional schema: TSchema;

```

```
optional schema: TSchema;

```[sync](#sync)ts

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```

```
sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```[utils](#utils)ts

```
utils: object;

```

```
utils: object;

```[utils.awaitTxId](#utilsawaittxid)ts

```
awaitTxId: AwaitTxIdFn;

```

```
awaitTxId: AwaitTxIdFn;

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/electric-db-collection/functions/electriccollectionoptions.md)[Electric DB Collection](/db/latest/docs/reference/electric-db-collection/index)[Query DB Collection](/db/latest/docs/reference/query-db-collection/index)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>