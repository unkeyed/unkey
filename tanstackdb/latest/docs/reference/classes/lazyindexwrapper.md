# LazyIndexWrapper | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: LazyIndexWrapper<TKey>Type ParametersConstructorsnew LazyIndexWrapper()ParametersidexpressionnameresolveroptionscollectionEntries?ReturnsMethodsgetExpression()ReturnsgetId()ReturnsgetName()ReturnsgetResolved()ReturnsisResolved()Returnsresolve()Returns# LazyIndexWrapper

Copy Markdown[Class: LazyIndexWrapper<TKey>](#class-lazyindexwrappertkey)

Defined in:[packages/db/src/indexes/lazy-index.ts:39](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L39)

Wrapper that defers index creation until first sync
[Type Parameters](#type-parameters)

•**TKey***extends*string|number=string|number
[Constructors](#constructors)[new LazyIndexWrapper()](#new-lazyindexwrapper)ts

```
new LazyIndexWrapper<TKey>(
   id, 
   expression, 
   name, 
   resolver, 
   options, 
collectionEntries?): LazyIndexWrapper<TKey>

```

```
new LazyIndexWrapper<TKey>(
   id, 
   expression, 
   name, 
   resolver, 
   options, 
collectionEntries?): LazyIndexWrapper<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:43](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L43)
[Parameters](#parameters)[id](#id)

number
[expression](#expression)

BasicExpression
[name](#name)

undefined|string
[resolver](#resolver)

[IndexResolver](/db/latest/docs/reference/type-aliases/indexresolver)<TKey>
[options](#options)

any
[collectionEntries?](#collectionentries)

Iterable<[TKey,any],any,any>
[Returns](#returns)

[LazyIndexWrapper](/db/latest/docs/reference/classes/lazyindexwrapper)<TKey>
[Methods](#methods)[getExpression()](#getexpression)ts

```
getExpression(): BasicExpression

```

```
getExpression(): BasicExpression

```

Defined in:[packages/db/src/indexes/lazy-index.ts:118](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L118)

Get the index expression
[Returns](#returns-1)

BasicExpression
[getId()](#getid)ts

```
getId(): number

```

```
getId(): number

```

Defined in:[packages/db/src/indexes/lazy-index.ts:104](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L104)

Get the index ID
[Returns](#returns-2)

number
[getName()](#getname)ts

```
getName(): undefined | string

```

```
getName(): undefined | string

```

Defined in:[packages/db/src/indexes/lazy-index.ts:111](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L111)

Get the index name
[Returns](#returns-3)

undefined|string
[getResolved()](#getresolved)ts

```
getResolved(): BaseIndex<TKey>

```

```
getResolved(): BaseIndex<TKey>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:92](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L92)

Get resolved index (throws if not ready)
[Returns](#returns-4)

[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>
[isResolved()](#isresolved)ts

```
isResolved(): boolean

```

```
isResolved(): boolean

```

Defined in:[packages/db/src/indexes/lazy-index.ts:85](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L85)

Check if already resolved
[Returns](#returns-5)

boolean
[resolve()](#resolve)ts

```
resolve(): Promise<BaseIndex<TKey>>

```

```
resolve(): Promise<BaseIndex<TKey>>

```

Defined in:[packages/db/src/indexes/lazy-index.ts:69](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/lazy-index.ts#L69)

Resolve the actual index
[Returns](#returns-6)

Promise<[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>>[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/lazyindexwrapper.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>