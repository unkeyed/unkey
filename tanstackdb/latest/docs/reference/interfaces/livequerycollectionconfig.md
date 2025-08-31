# LiveQueryCollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: LiveQueryCollectionConfig<TContext, TResult>ExampleType ParametersPropertiesgcTime?getKey()?ParametersitemReturnsid?onDelete?onInsert?onUpdate?queryschema?startSync?# LiveQueryCollectionConfig

Copy Markdown[Interface: LiveQueryCollectionConfig<TContext, TResult>](#interface-livequerycollectionconfigtcontext-tresult)

Defined in:[packages/db/src/query/live-query-collection.ts:47](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L47)

Configuration interface for live query collection options
[Example](#example)typescript

```
const config: LiveQueryCollectionConfig<any, any> = {
  // id is optional - will auto-generate "live-query-1", "live-query-2", etc.
  query: (q) => q
    .from({ comment: commentsCollection })
    .join(
      { user: usersCollection },
      ({ comment, user }) => eq(comment.user_id, user.id)
    )
    .where(({ comment }) => eq(comment.active, true))
    .select(({ comment, user }) => ({
      id: comment.id,
      content: comment.content,
      authorName: user.name,
    })),
  // getKey is optional - defaults to using stream key
  getKey: (item) => item.id,
}

```

```
const config: LiveQueryCollectionConfig<any, any> = {
  // id is optional - will auto-generate "live-query-1", "live-query-2", etc.
  query: (q) => q
    .from({ comment: commentsCollection })
    .join(
      { user: usersCollection },
      ({ comment, user }) => eq(comment.user_id, user.id)
    )
    .where(({ comment }) => eq(comment.active, true))
    .select(({ comment, user }) => ({
      id: comment.id,
      content: comment.content,
      authorName: user.name,
    })),
  // getKey is optional - defaults to using stream key
  getKey: (item) => item.id,
}

```[Type Parameters](#type-parameters)

•**TContext***extends*[Context](/db/latest/docs/reference/interfaces/context)

•**TResult***extends*object=[GetResult](/db/latest/docs/reference/type-aliases/getresult)<TContext> &object
[Properties](#properties)[gcTime?](#gctime)ts

```
optional gcTime: number;

```

```
optional gcTime: number;

```

Defined in:[packages/db/src/query/live-query-collection.ts:90](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L90)

GC time for the collection
[getKey()?](#getkey)ts

```
optional getKey: (item) => string | number;

```

```
optional getKey: (item) => string | number;

```

Defined in:[packages/db/src/query/live-query-collection.ts:68](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L68)

Function to extract the key from result items
If not provided, defaults to using the key from the D2 stream
[Parameters](#parameters)[item](#item)

TResult
[Returns](#returns)

string|number
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/query/live-query-collection.ts:55](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L55)

Unique identifier for the collection
If not provided, defaults tolive-query-${number}with auto-incrementing number
[onDelete?](#ondelete)ts

```
optional onDelete: DeleteMutationFn<TResult, string | number, Record<string, Fn>>;

```

```
optional onDelete: DeleteMutationFn<TResult, string | number, Record<string, Fn>>;

```

Defined in:[packages/db/src/query/live-query-collection.ts:80](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L80)
[onInsert?](#oninsert)ts

```
optional onInsert: InsertMutationFn<TResult, string | number, Record<string, Fn>>;

```

```
optional onInsert: InsertMutationFn<TResult, string | number, Record<string, Fn>>;

```

Defined in:[packages/db/src/query/live-query-collection.ts:78](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L78)

Optional mutation handlers
[onUpdate?](#onupdate)ts

```
optional onUpdate: UpdateMutationFn<TResult, string | number, Record<string, Fn>>;

```

```
optional onUpdate: UpdateMutationFn<TResult, string | number, Record<string, Fn>>;

```

Defined in:[packages/db/src/query/live-query-collection.ts:79](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L79)
[query](#query)ts

```
query: 
  | QueryBuilder<TContext>
| (q) => QueryBuilder<TContext>;

```

```
query: 
  | QueryBuilder<TContext>
| (q) => QueryBuilder<TContext>;

```

Defined in:[packages/db/src/query/live-query-collection.ts:60](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L60)

Query builder function that defines the live query
[schema?](#schema)ts

```
optional schema: StandardSchemaV1<unknown, unknown>;

```

```
optional schema: StandardSchemaV1<unknown, unknown>;

```

Defined in:[packages/db/src/query/live-query-collection.ts:73](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L73)

Optional schema for validation
[startSync?](#startsync)ts

```
optional startSync: boolean;

```

```
optional startSync: boolean;

```

Defined in:[packages/db/src/query/live-query-collection.ts:85](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L85)

Start sync / the query immediately[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/livequerycollectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>