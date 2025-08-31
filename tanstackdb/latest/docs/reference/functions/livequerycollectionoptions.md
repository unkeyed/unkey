# liveQueryCollectionOptions | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: liveQueryCollectionOptions()Type ParametersParametersconfigReturnsExample# liveQueryCollectionOptions

Copy Markdown[Function: liveQueryCollectionOptions()](#function-livequerycollectionoptions)ts

```
function liveQueryCollectionOptions<TContext, TResult>(config): CollectionConfig<TResult>

```

```
function liveQueryCollectionOptions<TContext, TResult>(config): CollectionConfig<TResult>

```

Defined in:[packages/db/src/query/live-query-collection.ts:117](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L117)

Creates live query collection options for use with createCollection
[Type Parameters](#type-parameters)

•**TContext***extends*[Context](/db/latest/docs/reference/interfaces/context)

•**TResult***extends*object= { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }
[Parameters](#parameters)[config](#config)

[LiveQueryCollectionConfig](/db/latest/docs/reference/interfaces/livequerycollectionconfig)<TContext,TResult>

Configuration options for the live query collection
[Returns](#returns)

[CollectionConfig](/db/latest/docs/reference/interfaces/collectionconfig)<TResult>

Collection options that can be passed to createCollection
[Example](#example)typescript

```
const options = liveQueryCollectionOptions({
  // id is optional - will auto-generate if not provided
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => eq(post.published, true))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      content: post.content,
    })),
  // getKey is optional - will use stream key if not provided
})

const collection = createCollection(options)

```

```
const options = liveQueryCollectionOptions({
  // id is optional - will auto-generate if not provided
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => eq(post.published, true))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      content: post.content,
    })),
  // getKey is optional - will use stream key if not provided
})

const collection = createCollection(options)

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/livequerycollectionoptions.md)[createCollection](/db/latest/docs/reference/functions/createcollection)[createLiveQueryCollection](/db/latest/docs/reference/functions/createlivequerycollection)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>