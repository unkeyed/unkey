# PendingMutation | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: PendingMutation<T, TOperation, TCollection>Type ParametersPropertieschangescollectioncreatedAtglobalKeykeymetadatamodifiedmutationIdoptimisticoriginalsyncMetadatatypeupdatedAt# PendingMutation

Copy Markdown[Interface: PendingMutation<T, TOperation, TCollection>](#interface-pendingmutationt-toperation-tcollection)

Defined in:[packages/db/src/types.ts:103](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L103)

Represents a pending mutation within a transaction
Contains information about the original and modified data, as well as metadata
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>

•**TOperation***extends*[OperationType](/db/latest/docs/reference/type-aliases/operationtype)=[OperationType](/db/latest/docs/reference/type-aliases/operationtype)

•**TCollection***extends*[Collection](/db/latest/docs/reference/interfaces/collection)<T,any,any,any,any> =[Collection](/db/latest/docs/reference/interfaces/collection)<T,any,any,any,any>
[Properties](#properties)[changes](#changes)ts

```
changes: ResolveTransactionChanges<T, TOperation>;

```

```
changes: ResolveTransactionChanges<T, TOperation>;

```

Defined in:[packages/db/src/types.ts:120](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L120)
[collection](#collection)ts

```
collection: TCollection;

```

```
collection: TCollection;

```

Defined in:[packages/db/src/types.ts:131](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L131)
[createdAt](#createdat)ts

```
createdAt: Date;

```

```
createdAt: Date;

```

Defined in:[packages/db/src/types.ts:129](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L129)
[globalKey](#globalkey)ts

```
globalKey: string;

```

```
globalKey: string;

```

Defined in:[packages/db/src/types.ts:121](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L121)
[key](#key)ts

```
key: any;

```

```
key: any;

```

Defined in:[packages/db/src/types.ts:123](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L123)
[metadata](#metadata)ts

```
metadata: unknown;

```

```
metadata: unknown;

```

Defined in:[packages/db/src/types.ts:125](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L125)
[modified](#modified)ts

```
modified: T;

```

```
modified: T;

```

Defined in:[packages/db/src/types.ts:118](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L118)
[mutationId](#mutationid)ts

```
mutationId: string;

```

```
mutationId: string;

```

Defined in:[packages/db/src/types.ts:114](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L114)
[optimistic](#optimistic)ts

```
optimistic: boolean;

```

```
optimistic: boolean;

```

Defined in:[packages/db/src/types.ts:128](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L128)

Whether this mutation should be applied optimistically (defaults to true)
[original](#original)ts

```
original: TOperation extends "insert" ? object : T;

```

```
original: TOperation extends "insert" ? object : T;

```

Defined in:[packages/db/src/types.ts:116](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L116)
[syncMetadata](#syncmetadata)ts

```
syncMetadata: Record<string, unknown>;

```

```
syncMetadata: Record<string, unknown>;

```

Defined in:[packages/db/src/types.ts:126](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L126)
[type](#type)ts

```
type: TOperation;

```

```
type: TOperation;

```

Defined in:[packages/db/src/types.ts:124](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L124)
[updatedAt](#updatedat)ts

```
updatedAt: Date;

```

```
updatedAt: Date;

```

Defined in:[packages/db/src/types.ts:130](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L130)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/pendingmutation.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>