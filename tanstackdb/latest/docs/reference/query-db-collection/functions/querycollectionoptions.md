# queryCollectionOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: queryCollectionOptions()Type ParametersParametersconfigReturns# queryCollectionOptions

Copy Markdown[Function: queryCollectionOptions()](#function-querycollectionoptions)ts

```
function queryCollectionOptions<TItem, TError, TQueryKey, TKey, TInsertInput>(config): CollectionConfig<TItem, string | number, StandardSchemaV1<unknown, unknown>, TItem> & object

```

```
function queryCollectionOptions<TItem, TError, TQueryKey, TKey, TInsertInput>(config): CollectionConfig<TItem, string | number, StandardSchemaV1<unknown, unknown>, TItem> & object

```

Defined in:[packages/query-db-collection/src/query.ts:277](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L277)

Creates query collection options for use with a standard Collection
[Type Parameters](#type-parameters)

•**TItem***extends*object

•**TError**=unknown

•**TQueryKey***extends*readonlyunknown[] = readonlyunknown[]

•**TKey***extends*string|number=string|number

•**TInsertInput***extends*object=TItem
[Parameters](#parameters)[config](#config)

[QueryCollectionConfig](/db/latest/docs/reference/query-db-collection/interfaces/querycollectionconfig)<TItem,TError,TQueryKey>

Configuration options for the Query collection
[Returns](#returns)

CollectionConfig<TItem,string|number,StandardSchemaV1<unknown,unknown>,TItem> &object

Collection options with utilities[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/query-db-collection/functions/querycollectionoptions.md)[Query DB Collection](/db/latest/docs/reference/query-db-collection/index)[React Hooks](/db/latest/docs/framework/react/reference/index)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>