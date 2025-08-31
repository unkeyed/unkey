# createLiveQueryCollection | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: createLiveQueryCollection()Call SignatureType ParametersParametersqueryReturnsExampleCall SignatureType ParametersParametersconfigReturnsExample# createLiveQueryCollection

Copy Markdown[Function: createLiveQueryCollection()](#function-createlivequerycollection)[Call Signature](#call-signature)ts

```
function createLiveQueryCollection<TContext, TResult>(query): Collection<TResult, string | number, {}>

```

```
function createLiveQueryCollection<TContext, TResult>(query): Collection<TResult, string | number, {}>

```

Defined in:[packages/db/src/query/live-query-collection.ts:424](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L424)

Creates a live query collection directly
[Type Parameters](#type-parameters)

•**TContext***extends*[Context](/db/latest/docs/reference/interfaces/context)

•**TResult***extends*object= { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }
[Parameters](#parameters)[query](#query)

(q) =>[QueryBuilder](/db/latest/docs/reference/type-aliases/querybuilder)<TContext>
[Returns](#returns)

[Collection](/db/latest/docs/reference/interfaces/collection)<TResult,string|number, {}>
[Example](#example)typescript

```
// Minimal usage - just pass a query function
const activeUsers = createLiveQueryCollection(
  (q) => q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({ id: user.id, name: user.name }))
)

// Full configuration with custom options
const searchResults = createLiveQueryCollection({
  id: "search-results", // Custom ID (auto-generated if omitted)
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => like(post.title, `%${searchTerm}%`))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      excerpt: post.excerpt,
    })),
  getKey: (item) => item.id, // Custom key function (uses stream key if omitted)
  utils: {
    updateSearchTerm: (newTerm: string) => {
      // Custom utility functions
    }
  }
})

```

```
// Minimal usage - just pass a query function
const activeUsers = createLiveQueryCollection(
  (q) => q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({ id: user.id, name: user.name }))
)

// Full configuration with custom options
const searchResults = createLiveQueryCollection({
  id: "search-results", // Custom ID (auto-generated if omitted)
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => like(post.title, `%${searchTerm}%`))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      excerpt: post.excerpt,
    })),
  getKey: (item) => item.id, // Custom key function (uses stream key if omitted)
  utils: {
    updateSearchTerm: (newTerm: string) => {
      // Custom utility functions
    }
  }
})

```[Call Signature](#call-signature-1)ts

```
function createLiveQueryCollection<TContext, TResult, TUtils>(config): Collection<TResult, string | number, TUtils>

```

```
function createLiveQueryCollection<TContext, TResult, TUtils>(config): Collection<TResult, string | number, TUtils>

```

Defined in:[packages/db/src/query/live-query-collection.ts:432](https://github.com/TanStack/db/blob/main/packages/db/src/query/live-query-collection.ts#L432)

Creates a live query collection directly
[Type Parameters](#type-parameters-1)

•**TContext***extends*[Context](/db/latest/docs/reference/interfaces/context)

•**TResult***extends*object= { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }

•**TUtils***extends*[UtilsRecord](/db/latest/docs/reference/type-aliases/utilsrecord)= {}
[Parameters](#parameters-1)[config](#config)

[LiveQueryCollectionConfig](/db/latest/docs/reference/interfaces/livequerycollectionconfig)<TContext,TResult> &object
[Returns](#returns-1)

[Collection](/db/latest/docs/reference/interfaces/collection)<TResult,string|number,TUtils>
[Example](#example-1)typescript

```
// Minimal usage - just pass a query function
const activeUsers = createLiveQueryCollection(
  (q) => q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({ id: user.id, name: user.name }))
)

// Full configuration with custom options
const searchResults = createLiveQueryCollection({
  id: "search-results", // Custom ID (auto-generated if omitted)
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => like(post.title, `%${searchTerm}%`))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      excerpt: post.excerpt,
    })),
  getKey: (item) => item.id, // Custom key function (uses stream key if omitted)
  utils: {
    updateSearchTerm: (newTerm: string) => {
      // Custom utility functions
    }
  }
})

```

```
// Minimal usage - just pass a query function
const activeUsers = createLiveQueryCollection(
  (q) => q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({ id: user.id, name: user.name }))
)

// Full configuration with custom options
const searchResults = createLiveQueryCollection({
  id: "search-results", // Custom ID (auto-generated if omitted)
  query: (q) => q
    .from({ post: postsCollection })
    .where(({ post }) => like(post.title, `%${searchTerm}%`))
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      excerpt: post.excerpt,
    })),
  getKey: (item) => item.id, // Custom key function (uses stream key if omitted)
  utils: {
    updateSearchTerm: (newTerm: string) => {
      // Custom utility functions
    }
  }
})

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/createlivequerycollection.md)[liveQueryCollectionOptions](/db/latest/docs/reference/functions/livequerycollectionoptions)[createOptimisticAction](/db/latest/docs/reference/functions/createoptimisticaction)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>