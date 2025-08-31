# QueryCollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: QueryCollectionConfig<TItem, TError, TQueryKey>Type ParametersPropertiesenabled?getKey()ParametersitemReturnsid?meta?ExampleonDelete?ParamReturnsExamplesonInsert?ParamReturnsExamplesonUpdate?ParamReturnsExamplesqueryClientqueryFn()Parameterscontextclientdirection?metapageParam?queryKeysignalReturnsqueryKeyrefetchInterval?retry?retryDelay?schema?staleTime?startSync?sync?# QueryCollectionConfig

Copy Markdown[Interface: QueryCollectionConfig<TItem, TError, TQueryKey>](#interface-querycollectionconfigtitem-terror-tquerykey)

Defined in:[packages/query-db-collection/src/query.ts:32](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L32)
[Type Parameters](#type-parameters)

•**TItem***extends*object

•**TError**=unknown

•**TQueryKey***extends*QueryKey=QueryKey
[Properties](#properties)[enabled?](#enabled)ts

```
optional enabled: boolean;

```

```
optional enabled: boolean;

```

Defined in:[packages/query-db-collection/src/query.ts:42](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L42)
[getKey()](#getkey)ts

```
getKey: (item) => string | number;

```

```
getKey: (item) => string | number;

```

Defined in:[packages/query-db-collection/src/query.ts:74](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L74)
[Parameters](#parameters)[item](#item)

TItem
[Returns](#returns)

string|number
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/query-db-collection/src/query.ts:73](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L73)
[meta?](#meta)ts

```
optional meta: Record<string, unknown>;

```

```
optional meta: Record<string, unknown>;

```

Defined in:[packages/query-db-collection/src/query.ts:242](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L242)

Metadata to pass to the query.
Available in queryFn via context.meta
[Example](#example)ts

```
// Using meta for error context
queryFn: async (context) => {
  try {
    return await api.getTodos(userId)
  } catch (error) {
    // Use meta for better error messages
    throw new Error(
      context.meta?.errorMessage || 'Failed to load todos'
    )
  }
},
meta: {
  errorMessage: `Failed to load todos for user ${userId}`
}

```

```
// Using meta for error context
queryFn: async (context) => {
  try {
    return await api.getTodos(userId)
  } catch (error) {
    // Use meta for better error messages
    throw new Error(
      context.meta?.errorMessage || 'Failed to load todos'
    )
  }
},
meta: {
  errorMessage: `Failed to load todos for user ${userId}`
}

```[onDelete?](#ondelete)ts

```
optional onDelete: DeleteMutationFn<TItem>;

```

```
optional onDelete: DeleteMutationFn<TItem>;

```

Defined in:[packages/query-db-collection/src/query.ts:219](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L219)

Optional asynchronous handler function called before a delete operation
[Param](#param)

Object containing transaction and collection information
[Returns](#returns-1)

Promise resolving to void or { refetch?: boolean } to control refetching
[Examples](#examples)ts

```
// Basic query collection delete handler
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)
  // Automatically refetches query after delete
}

```

```
// Basic query collection delete handler
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)
  // Automatically refetches query after delete
}

```ts

```
// Delete handler with refetch control
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)
  return { refetch: false } // Skip automatic refetch
}

```

```
// Delete handler with refetch control
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)
  return { refetch: false } // Skip automatic refetch
}

```ts

```
// Delete handler with multiple items
onDelete: async ({ transaction }) => {
  const keysToDelete = transaction.mutations.map(m => m.key)
  await api.deleteTodos(keysToDelete)
  // Will refetch query to get updated data
}

```

```
// Delete handler with multiple items
onDelete: async ({ transaction }) => {
  const keysToDelete = transaction.mutations.map(m => m.key)
  await api.deleteTodos(keysToDelete)
  // Will refetch query to get updated data
}

```ts

```
// Delete handler with related collection refetch
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)

  // Refetch related collections when this item is deleted
  await Promise.all([
    collection.utils.refetch(), // Refetch this collection
    usersCollection.utils.refetch(), // Refetch users
    projectsCollection.utils.refetch() // Refetch projects
  ])

  return { refetch: false } // Skip automatic refetch since we handled it manually
}

```

```
// Delete handler with related collection refetch
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.deleteTodo(mutation.original.id)

  // Refetch related collections when this item is deleted
  await Promise.all([
    collection.utils.refetch(), // Refetch this collection
    usersCollection.utils.refetch(), // Refetch users
    projectsCollection.utils.refetch() // Refetch projects
  ])

  return { refetch: false } // Skip automatic refetch since we handled it manually
}

```[onInsert?](#oninsert)ts

```
optional onInsert: InsertMutationFn<TItem>;

```

```
optional onInsert: InsertMutationFn<TItem>;

```

Defined in:[packages/query-db-collection/src/query.ts:120](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L120)

Optional asynchronous handler function called before an insert operation
[Param](#param-1)

Object containing transaction and collection information
[Returns](#returns-2)

Promise resolving to void or { refetch?: boolean } to control refetching
[Examples](#examples-1)ts

```
// Basic query collection insert handler
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
  // Automatically refetches query after insert
}

```

```
// Basic query collection insert handler
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
  // Automatically refetches query after insert
}

```ts

```
// Insert handler with refetch control
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
  return { refetch: false } // Skip automatic refetch
}

```

```
// Insert handler with refetch control
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
  return { refetch: false } // Skip automatic refetch
}

```ts

```
// Insert handler with multiple items
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  await api.createTodos(items)
  // Will refetch query to get updated data
}

```

```
// Insert handler with multiple items
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  await api.createTodos(items)
  // Will refetch query to get updated data
}

```ts

```
// Insert handler with error handling
onInsert: async ({ transaction }) => {
  try {
    const newItem = transaction.mutations[0].modified
    await api.createTodo(newItem)
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // Transaction will rollback optimistic changes
  }
}

```

```
// Insert handler with error handling
onInsert: async ({ transaction }) => {
  try {
    const newItem = transaction.mutations[0].modified
    await api.createTodo(newItem)
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // Transaction will rollback optimistic changes
  }
}

```[onUpdate?](#onupdate)ts

```
optional onUpdate: UpdateMutationFn<TItem>;

```

```
optional onUpdate: UpdateMutationFn<TItem>;

```

Defined in:[packages/query-db-collection/src/query.ts:173](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L173)

Optional asynchronous handler function called before an update operation
[Param](#param-2)

Object containing transaction and collection information
[Returns](#returns-3)

Promise resolving to void or { refetch?: boolean } to control refetching
[Examples](#examples-2)ts

```
// Basic query collection update handler
onUpdate: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)
  // Automatically refetches query after update
}

```

```
// Basic query collection update handler
onUpdate: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)
  // Automatically refetches query after update
}

```ts

```
// Update handler with multiple items
onUpdate: async ({ transaction }) => {
  const updates = transaction.mutations.map(m => ({
    id: m.key,
    changes: m.changes
  }))
  await api.updateTodos(updates)
  // Will refetch query to get updated data
}

```

```
// Update handler with multiple items
onUpdate: async ({ transaction }) => {
  const updates = transaction.mutations.map(m => ({
    id: m.key,
    changes: m.changes
  }))
  await api.updateTodos(updates)
  // Will refetch query to get updated data
}

```ts

```
// Update handler with manual refetch
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)

  // Manually trigger refetch
  await collection.utils.refetch()

  return { refetch: false } // Skip automatic refetch
}

```

```
// Update handler with manual refetch
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)

  // Manually trigger refetch
  await collection.utils.refetch()

  return { refetch: false } // Skip automatic refetch
}

```ts

```
// Update handler with related collection refetch
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)

  // Refetch related collections when this item changes
  await Promise.all([
    collection.utils.refetch(), // Refetch this collection
    usersCollection.utils.refetch(), // Refetch users
    tagsCollection.utils.refetch() // Refetch tags
  ])

  return { refetch: false } // Skip automatic refetch since we handled it manually
}

```

```
// Update handler with related collection refetch
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.updateTodo(mutation.original.id, mutation.changes)

  // Refetch related collections when this item changes
  await Promise.all([
    collection.utils.refetch(), // Refetch this collection
    usersCollection.utils.refetch(), // Refetch users
    tagsCollection.utils.refetch() // Refetch tags
  ])

  return { refetch: false } // Skip automatic refetch since we handled it manually
}

```[queryClient](#queryclient)ts

```
queryClient: QueryClient;

```

```
queryClient: QueryClient;

```

Defined in:[packages/query-db-collection/src/query.ts:39](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L39)
[queryFn()](#queryfn)ts

```
queryFn: (context) => Promise<TItem[]>;

```

```
queryFn: (context) => Promise<TItem[]>;

```

Defined in:[packages/query-db-collection/src/query.ts:38](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L38)
[Parameters](#parameters-1)[context](#context)[client](#client)

QueryClient
[direction?](#direction)

unknown

**Deprecated**

if you want access to the direction, you can add it to the pageParam
[meta](#meta-1)

undefined|Record<string,unknown>
[pageParam?](#pageparam)

unknown
[queryKey](#querykey)

TQueryKey
[signal](#signal)

AbortSignal
[Returns](#returns-4)

Promise<TItem[]>
[queryKey](#querykey-1)ts

```
queryKey: TQueryKey;

```

```
queryKey: TQueryKey;

```

Defined in:[packages/query-db-collection/src/query.ts:37](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L37)
[refetchInterval?](#refetchinterval)ts

```
optional refetchInterval: number | false | (query) => undefined | number | false;

```

```
optional refetchInterval: number | false | (query) => undefined | number | false;

```

Defined in:[packages/query-db-collection/src/query.ts:43](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L43)
[retry?](#retry)ts

```
optional retry: RetryValue<TError>;

```

```
optional retry: RetryValue<TError>;

```

Defined in:[packages/query-db-collection/src/query.ts:50](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L50)
[retryDelay?](#retrydelay)ts

```
optional retryDelay: RetryDelayValue<TError>;

```

```
optional retryDelay: RetryDelayValue<TError>;

```

Defined in:[packages/query-db-collection/src/query.ts:57](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L57)
[schema?](#schema)ts

```
optional schema: StandardSchemaV1<unknown, unknown>;

```

```
optional schema: StandardSchemaV1<unknown, unknown>;

```

Defined in:[packages/query-db-collection/src/query.ts:75](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L75)
[staleTime?](#staletime)ts

```
optional staleTime: StaleTimeFunction<TItem[], TError, TItem[], TQueryKey>;

```

```
optional staleTime: StaleTimeFunction<TItem[], TError, TItem[], TQueryKey>;

```

Defined in:[packages/query-db-collection/src/query.ts:64](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L64)
[startSync?](#startsync)ts

```
optional startSync: boolean;

```

```
optional startSync: boolean;

```

Defined in:[packages/query-db-collection/src/query.ts:77](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L77)
[sync?](#sync)ts

```
optional sync: SyncConfig<TItem, string | number>;

```

```
optional sync: SyncConfig<TItem, string | number>;

```

Defined in:[packages/query-db-collection/src/query.ts:76](https://github.com/TanStack/db/blob/main/packages/query-db-collection/src/query.ts#L76)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/query-db-collection/interfaces/querycollectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>