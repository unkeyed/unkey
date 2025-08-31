# QueryCollectionUtils | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: QueryCollectionUtils<TItem, TKey, TInsertInput>ExtendsType ParametersIndexablePropertiesrefetchwriteBatch()ParametersoperationsReturnswriteDelete()ParameterskeysReturnswriteInsert()ParametersdataReturnswriteUpdate()ParametersupdatesReturnswriteUpsert()ParametersdataReturns# QueryCollectionUtils

Copy Markdown[Interface: QueryCollectionUtils<TItem, TKey, TInsertInput>](#interface-querycollectionutilstitem-tkey-tinsertinput)

Defined in:[packages/query-db-collection/src/query.ts:256](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L256)

Write operation types for batch operations
[Extends](#extends)

- UtilsRecord
[Type Parameters](#type-parameters)

•**TItem***extends*object=Record<string,unknown>

•**TKey***extends*string|number=string|number

•**TInsertInput***extends*object=TItem
[Indexable](#indexable)ts

```
[key: string]: Fn

```

```
[key: string]: Fn

```[Properties](#properties)[refetch](#refetch)ts

```
refetch: RefetchFn;

```

```
refetch: RefetchFn;

```

Defined in:[packages/query-db-collection/src/query.ts:261](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L261)
[writeBatch()](#writebatch)ts

```
writeBatch: (operations) => void;

```

```
writeBatch: (operations) => void;

```

Defined in:[packages/query-db-collection/src/query.ts:266](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L266)
[Parameters](#parameters)[operations](#operations)

[SyncOperation](/db/latest/docs/reference/query-db-collection/type-aliases/syncoperation)<TItem,TKey,TInsertInput>[]
[Returns](#returns)

void
[writeDelete()](#writedelete)ts

```
writeDelete: (keys) => void;

```

```
writeDelete: (keys) => void;

```

Defined in:[packages/query-db-collection/src/query.ts:264](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L264)
[Parameters](#parameters-1)[keys](#keys)

TKey|TKey[]
[Returns](#returns-1)

void
[writeInsert()](#writeinsert)ts

```
writeInsert: (data) => void;

```

```
writeInsert: (data) => void;

```

Defined in:[packages/query-db-collection/src/query.ts:262](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L262)
[Parameters](#parameters-2)[data](#data)

TInsertInput|TInsertInput[]
[Returns](#returns-2)

void
[writeUpdate()](#writeupdate)ts

```
writeUpdate: (updates) => void;

```

```
writeUpdate: (updates) => void;

```

Defined in:[packages/query-db-collection/src/query.ts:263](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L263)
[Parameters](#parameters-3)[updates](#updates)

Partial<TItem> |Partial<TItem>[]
[Returns](#returns-3)

void
[writeUpsert()](#writeupsert)ts

```
writeUpsert: (data) => void;

```

```
writeUpsert: (data) => void;

```

Defined in:[packages/query-db-collection/src/query.ts:265](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L265)
[Parameters](#parameters-4)[data](#data-1)

Partial<TItem> |Partial<TItem>[]
[Returns](#returns-4)

void[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/query-db-collection/interfaces/querycollectionutils.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>