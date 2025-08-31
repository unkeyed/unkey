# IndexProxy | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: IndexProxy<TKey>Type ParametersConstructorsnew IndexProxy()ParametersindexIdlazyIndexReturnsAccessorsexpressionGet SignatureReturnsidGet SignatureReturnsindexGet SignatureReturnsindexedKeysSetGet SignatureReturnsisReadyGet SignatureReturnskeyCountGet SignatureReturnsnameGet SignatureReturnsorderedEntriesArrayGet SignatureReturnsvalueMapDataGet SignatureReturnsMethods_getLazyWrapper()ReturnsequalityLookup()ParametersvalueReturnsgetStats()ReturnsinArrayLookup()ParametersvaluesReturnsmatchesField()ParametersfieldPathReturnsrangeQuery()ParametersoptionsReturnssupports()ParametersoperationReturnswhenReady()Returns# IndexProxy

Copy Markdown[Class: IndexProxy<TKey>](#class-indexproxytkey)

Defined in:[packages/db/src/indexes/lazy-index.ts:131](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L131)

Proxy that provides synchronous interface while index loads asynchronously
[Type Parameters](#type-parameters)

•**TKey***extends*string|number=string|number
[Constructors](#constructors)[new IndexProxy()](#new-indexproxy)ts

```
new IndexProxy<TKey>(indexId, lazyIndex): IndexProxy<TKey>

```

```
new IndexProxy<TKey>(indexId, lazyIndex): IndexProxy<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:132](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L132)
[Parameters](#parameters)[indexId](#indexid)

number
[lazyIndex](#lazyindex)

[LazyIndexWrapper](/db/latest/docs/reference/classes/lazyindexwrapper)<TKey>
[Returns](#returns)

[IndexProxy](/db/latest/docs/reference/classes/indexproxy)<TKey>
[Accessors](#accessors)[expression](#expression)[Get Signature](#get-signature)ts

```
get expression(): BasicExpression

```

```
get expression(): BasicExpression

```

Defined in:[packages/db/src/indexes/lazy-index.ts:178](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L178)

Get the index expression (available immediately)
[Returns](#returns-1)

BasicExpression
[id](#id)[Get Signature](#get-signature-1)ts

```
get id(): number

```

```
get id(): number

```

Defined in:[packages/db/src/indexes/lazy-index.ts:161](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L161)

Get the index ID
[Returns](#returns-2)

number
[index](#index)[Get Signature](#get-signature-2)ts

```
get index(): BaseIndex<TKey>

```

```
get index(): BaseIndex<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:140](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L140)

Get the resolved index (throws if not ready)
[Returns](#returns-3)

[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>
[indexedKeysSet](#indexedkeysset)[Get Signature](#get-signature-3)ts

```
get indexedKeysSet(): Set<TKey>

```

```
get indexedKeysSet(): Set<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:216](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L216)
[Returns](#returns-4)

Set<TKey>
[isReady](#isready)[Get Signature](#get-signature-4)ts

```
get isReady(): boolean

```

```
get isReady(): boolean

```

Defined in:[packages/db/src/indexes/lazy-index.ts:147](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L147)

Check if index is ready
[Returns](#returns-5)

boolean
[keyCount](#keycount)[Get Signature](#get-signature-5)ts

```
get keyCount(): number

```

```
get keyCount(): number

```

Defined in:[packages/db/src/indexes/lazy-index.ts:211](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L211)

Get the key count (throws if not ready)
[Returns](#returns-6)

number
[name](#name)[Get Signature](#get-signature-6)ts

```
get name(): undefined | string

```

```
get name(): undefined | string

```

Defined in:[packages/db/src/indexes/lazy-index.ts:168](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L168)

Get the index name (throws if not ready)
[Returns](#returns-7)

undefined|string
[orderedEntriesArray](#orderedentriesarray)[Get Signature](#get-signature-7)ts

```
get orderedEntriesArray(): [any, Set<TKey>][]

```

```
get orderedEntriesArray(): [any, Set<TKey>][]

```

Defined in:[packages/db/src/indexes/lazy-index.ts:221](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L221)
[Returns](#returns-8)

[any,Set<TKey>][]
[valueMapData](#valuemapdata)[Get Signature](#get-signature-8)ts

```
get valueMapData(): Map<any, Set<TKey>>

```

```
get valueMapData(): Map<any, Set<TKey>>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:226](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L226)
[Returns](#returns-9)

Map<any,Set<TKey>>
[Methods](#methods)[_getLazyWrapper()](#_getlazywrapper)ts

```
_getLazyWrapper(): LazyIndexWrapper<TKey>

```

```
_getLazyWrapper(): LazyIndexWrapper<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:248](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L248)
[Returns](#returns-10)

[LazyIndexWrapper](/db/latest/docs/reference/classes/lazyindexwrapper)<TKey>
[equalityLookup()](#equalitylookup)ts

```
equalityLookup(value): Set<TKey>

```

```
equalityLookup(value): Set<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:232](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L232)
[Parameters](#parameters-1)[value](#value)

any
[Returns](#returns-11)

Set<TKey>
[getStats()](#getstats)ts

```
getStats(): IndexStats

```

```
getStats(): IndexStats

```

Defined in:[packages/db/src/indexes/lazy-index.ts:192](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L192)

Get index statistics (throws if not ready)
[Returns](#returns-12)

[IndexStats](/db/latest/docs/reference/interfaces/indexstats)
[inArrayLookup()](#inarraylookup)ts

```
inArrayLookup(values): Set<TKey>

```

```
inArrayLookup(values): Set<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:242](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L242)
[Parameters](#parameters-2)[values](#values)

any[]
[Returns](#returns-13)

Set<TKey>
[matchesField()](#matchesfield)ts

```
matchesField(fieldPath): boolean

```

```
matchesField(fieldPath): boolean

```

Defined in:[packages/db/src/indexes/lazy-index.ts:199](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L199)

Check if index matches a field path (available immediately)
[Parameters](#parameters-3)[fieldPath](#fieldpath)

string[]
[Returns](#returns-14)

boolean
[rangeQuery()](#rangequery)ts

```
rangeQuery(options): Set<TKey>

```

```
rangeQuery(options): Set<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:237](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L237)
[Parameters](#parameters-4)[options](#options)

any
[Returns](#returns-15)

Set<TKey>
[supports()](#supports)ts

```
supports(operation): boolean

```

```
supports(operation): boolean

```

Defined in:[packages/db/src/indexes/lazy-index.ts:185](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L185)

Check if index supports an operation (throws if not ready)
[Parameters](#parameters-5)[operation](#operation)

any
[Returns](#returns-16)

boolean
[whenReady()](#whenready)ts

```
whenReady(): Promise<BaseIndex<TKey>>

```

```
whenReady(): Promise<BaseIndex<TKey>>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:154](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L154)

Wait for index to be ready
[Returns](#returns-17)

Promise<[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>>[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/indexproxy.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>