# TransactionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: TransactionConfig<T>Type ParametersPropertiesautoCommit?id?metadata?mutationFn# TransactionConfig

Copy Markdown[Interface: TransactionConfig<T>](#interface-transactionconfigt)

Defined in:[packages/db/src/types.ts:161](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L161)
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>
[Properties](#properties)[autoCommit?](#autocommit)ts

```
optional autoCommit: boolean;

```

```
optional autoCommit: boolean;

```

Defined in:[packages/db/src/types.ts:165](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L165)
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/types.ts:163](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L163)

Unique identifier for the transaction
[metadata?](#metadata)ts

```
optional metadata: Record<string, unknown>;

```

```
optional metadata: Record<string, unknown>;

```

Defined in:[packages/db/src/types.ts:168](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L168)

Custom metadata to associate with the transaction
[mutationFn](#mutationfn)ts

```
mutationFn: MutationFn<T>;

```

```
mutationFn: MutationFn<T>;

```

Defined in:[packages/db/src/types.ts:166](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L166)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/transactionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>