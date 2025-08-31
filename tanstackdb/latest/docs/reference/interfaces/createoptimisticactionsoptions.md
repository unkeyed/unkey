# CreateOptimisticActionsOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: CreateOptimisticActionsOptions<TVars, T>ExtendsType ParametersPropertiesautoCommit?Inherited fromid?Inherited frommetadata?Inherited frommutationFn()ParametersvarsparamsReturnsonMutate()ParametersvarsReturns# CreateOptimisticActionsOptions

Copy Markdown[Interface: CreateOptimisticActionsOptions<TVars, T>](#interface-createoptimisticactionsoptionstvars-t)

Defined in:[packages/db/src/types.ts:174](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L174)

Options for the createOptimisticAction helper
[Extends](#extends)

- Omit<[TransactionConfig](/db/latest/docs/reference/interfaces/transactionconfig)<T>,"mutationFn">
[Type Parameters](#type-parameters)

•**TVars**=unknown

•**T***extends*object=Record<string,unknown>
[Properties](#properties)[autoCommit?](#autocommit)ts

```
optional autoCommit: boolean;

```

```
optional autoCommit: boolean;

```

Defined in:[packages/db/src/types.ts:165](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L165)
[Inherited from](#inherited-from)ts

```
Omit.autoCommit

```

```
Omit.autoCommit

```[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/types.ts:163](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L163)

Unique identifier for the transaction
[Inherited from](#inherited-from-1)ts

```
Omit.id

```

```
Omit.id

```[metadata?](#metadata)ts

```
optional metadata: Record<string, unknown>;

```

```
optional metadata: Record<string, unknown>;

```

Defined in:[packages/db/src/types.ts:168](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L168)

Custom metadata to associate with the transaction
[Inherited from](#inherited-from-2)ts

```
Omit.metadata

```

```
Omit.metadata

```[mutationFn()](#mutationfn)ts

```
mutationFn: (vars, params) => Promise<any>;

```

```
mutationFn: (vars, params) => Promise<any>;

```

Defined in:[packages/db/src/types.ts:181](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L181)

Function to execute the mutation on the server
[Parameters](#parameters)[vars](#vars)

TVars
[params](#params)

[MutationFnParams](/db/latest/docs/reference/type-aliases/mutationfnparams)<T>
[Returns](#returns)

Promise<any>
[onMutate()](#onmutate)ts

```
onMutate: (vars) => void;

```

```
onMutate: (vars) => void;

```

Defined in:[packages/db/src/types.ts:179](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L179)

Function to apply optimistic updates locally before the mutation completes
[Parameters](#parameters-1)[vars](#vars-1)

TVars
[Returns](#returns-1)

void[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/createoptimisticactionsoptions.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>