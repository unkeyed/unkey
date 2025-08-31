# CollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: CollectionConfig<T, TKey, TSchema, TInsertInput>Type ParametersPropertiesautoIndex?DefaultDescriptioncompare()?ParametersxyReturnsExamplegcTime?getKey()ParametersitemReturnsExampleid?onDelete?ParamReturnsExamplesonInsert?ParamReturnsExamplesonUpdate?ParamReturnsExamplesschema?startSync?sync# CollectionConfig

Copy Markdown[Interface: CollectionConfig<T, TKey, TSchema, TInsertInput>](#interface-collectionconfigt-tkey-tschema-tinsertinput)

Defined in:[packages/db/src/types.ts:349](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L349)
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>

•**TKey***extends*string|number=string|number

•**TSchema***extends*StandardSchemaV1=StandardSchemaV1

•**TInsertInput***extends*object=T
[Properties](#properties)[autoIndex?](#autoindex)ts

```
optional autoIndex: "off" | "eager";

```

```
optional autoIndex: "off" | "eager";

```

Defined in:[packages/db/src/types.ts:388](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L388)

Auto-indexing mode for the collection.
When enabled, indexes will be automatically created for simple where expressions.
[Default](#default)ts

```
"eager"

```

```
"eager"

```[Description](#description)

- "off": No automatic indexing
- "eager": Automatically create indexes for simple where expressions in subscribeChanges (default)
[compare()?](#compare)ts

```
optional compare: (x, y) => number;

```

```
optional compare: (x, y) => number;

```

Defined in:[packages/db/src/types.ts:399](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L399)

Optional function to compare two items.
This is used to order the items in the collection.
[Parameters](#parameters)[x](#x)

T

The first item to compare
[y](#y)

T

The second item to compare
[Returns](#returns)

number

A number indicating the order of the items
[Example](#example)ts

```
// For a collection with a 'createdAt' field
compare: (x, y) => x.createdAt.getTime() - y.createdAt.getTime()

```

```
// For a collection with a 'createdAt' field
compare: (x, y) => x.createdAt.getTime() - y.createdAt.getTime()

```[gcTime?](#gctime)ts

```
optional gcTime: number;

```

```
optional gcTime: number;

```

Defined in:[packages/db/src/types.ts:374](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L374)

Time in milliseconds after which the collection will be garbage collected
when it has no active subscribers. Defaults to 5 minutes (300000ms).
[getKey()](#getkey)ts

```
getKey: (item) => TKey;

```

```
getKey: (item) => TKey;

```

Defined in:[packages/db/src/types.ts:369](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L369)

Function to extract the ID from an object
This is required for update/delete operations which now only accept IDs
[Parameters](#parameters-1)[item](#item)

T

The item to extract the ID from
[Returns](#returns-1)

TKey

The ID string for the item
[Example](#example-1)ts

```
// For a collection with a 'uuid' field as the primary key
getKey: (item) => item.uuid

```

```
// For a collection with a 'uuid' field as the primary key
getKey: (item) => item.uuid

```[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/db/src/types.ts:357](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L357)
[onDelete?](#ondelete)ts

```
optional onDelete: DeleteMutationFn<T, TKey, Record<string, Fn>>;

```

```
optional onDelete: DeleteMutationFn<T, TKey, Record<string, Fn>>;

```

Defined in:[packages/db/src/types.ts:528](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L528)

Optional asynchronous handler function called before a delete operation
[Param](#param)

Object containing transaction and collection information
[Returns](#returns-2)

Promise resolving to any value
[Examples](#examples)ts

```
// Basic delete handler
onDelete: async ({ transaction, collection }) => {
  const deletedKey = transaction.mutations[0].key
  await api.deleteTodo(deletedKey)
}

```

```
// Basic delete handler
onDelete: async ({ transaction, collection }) => {
  const deletedKey = transaction.mutations[0].key
  await api.deleteTodo(deletedKey)
}

```ts

```
// Delete handler with multiple items
onDelete: async ({ transaction, collection }) => {
  const keysToDelete = transaction.mutations.map(m => m.key)
  await api.deleteTodos(keysToDelete)
}

```

```
// Delete handler with multiple items
onDelete: async ({ transaction, collection }) => {
  const keysToDelete = transaction.mutations.map(m => m.key)
  await api.deleteTodos(keysToDelete)
}

```ts

```
// Delete handler with confirmation
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  const shouldDelete = await confirmDeletion(mutation.original)
  if (!shouldDelete) {
    throw new Error('Delete cancelled by user')
  }
  await api.deleteTodo(mutation.original.id)
}

```

```
// Delete handler with confirmation
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  const shouldDelete = await confirmDeletion(mutation.original)
  if (!shouldDelete) {
    throw new Error('Delete cancelled by user')
  }
  await api.deleteTodo(mutation.original.id)
}

```ts

```
// Delete handler with optimistic rollback
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  try {
    await api.deleteTodo(mutation.original.id)
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Delete failed, rolling back:', error)
    throw error
  }
}

```

```
// Delete handler with optimistic rollback
onDelete: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  try {
    await api.deleteTodo(mutation.original.id)
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Delete failed, rolling back:', error)
    throw error
  }
}

```[onInsert?](#oninsert)ts

```
optional onInsert: InsertMutationFn<TInsertInput, TKey, Record<string, Fn>>;

```

```
optional onInsert: InsertMutationFn<TInsertInput, TKey, Record<string, Fn>>;

```

Defined in:[packages/db/src/types.ts:441](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L441)

Optional asynchronous handler function called before an insert operation
[Param](#param-1)

Object containing transaction and collection information
[Returns](#returns-3)

Promise resolving to any value
[Examples](#examples-1)ts

```
// Basic insert handler
onInsert: async ({ transaction, collection }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
}

```

```
// Basic insert handler
onInsert: async ({ transaction, collection }) => {
  const newItem = transaction.mutations[0].modified
  await api.createTodo(newItem)
}

```ts

```
// Insert handler with multiple items
onInsert: async ({ transaction, collection }) => {
  const items = transaction.mutations.map(m => m.modified)
  await api.createTodos(items)
}

```

```
// Insert handler with multiple items
onInsert: async ({ transaction, collection }) => {
  const items = transaction.mutations.map(m => m.modified)
  await api.createTodos(items)
}

```ts

```
// Insert handler with error handling
onInsert: async ({ transaction, collection }) => {
  try {
    const newItem = transaction.mutations[0].modified
    const result = await api.createTodo(newItem)
    return result
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // This will cause the transaction to fail
  }
}

```

```
// Insert handler with error handling
onInsert: async ({ transaction, collection }) => {
  try {
    const newItem = transaction.mutations[0].modified
    const result = await api.createTodo(newItem)
    return result
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // This will cause the transaction to fail
  }
}

```ts

```
// Insert handler with metadata
onInsert: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.createTodo(mutation.modified, {
    source: mutation.metadata?.source,
    timestamp: mutation.createdAt
  })
}

```

```
// Insert handler with metadata
onInsert: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  await api.createTodo(mutation.modified, {
    source: mutation.metadata?.source,
    timestamp: mutation.createdAt
  })
}

```[onUpdate?](#onupdate)ts

```
optional onUpdate: UpdateMutationFn<T, TKey, Record<string, Fn>>;

```

```
optional onUpdate: UpdateMutationFn<T, TKey, Record<string, Fn>>;

```

Defined in:[packages/db/src/types.ts:485](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L485)

Optional asynchronous handler function called before an update operation
[Param](#param-2)

Object containing transaction and collection information
[Returns](#returns-4)

Promise resolving to any value
[Examples](#examples-2)ts

```
// Basic update handler
onUpdate: async ({ transaction, collection }) => {
  const updatedItem = transaction.mutations[0].modified
  await api.updateTodo(updatedItem.id, updatedItem)
}

```

```
// Basic update handler
onUpdate: async ({ transaction, collection }) => {
  const updatedItem = transaction.mutations[0].modified
  await api.updateTodo(updatedItem.id, updatedItem)
}

```ts

```
// Update handler with partial updates
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  const changes = mutation.changes // Only the changed fields
  await api.updateTodo(mutation.original.id, changes)
}

```

```
// Update handler with partial updates
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  const changes = mutation.changes // Only the changed fields
  await api.updateTodo(mutation.original.id, changes)
}

```ts

```
// Update handler with multiple items
onUpdate: async ({ transaction, collection }) => {
  const updates = transaction.mutations.map(m => ({
    id: m.key,
    changes: m.changes
  }))
  await api.updateTodos(updates)
}

```

```
// Update handler with multiple items
onUpdate: async ({ transaction, collection }) => {
  const updates = transaction.mutations.map(m => ({
    id: m.key,
    changes: m.changes
  }))
  await api.updateTodos(updates)
}

```ts

```
// Update handler with optimistic rollback
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  try {
    await api.updateTodo(mutation.original.id, mutation.changes)
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Update failed, rolling back:', error)
    throw error
  }
}

```

```
// Update handler with optimistic rollback
onUpdate: async ({ transaction, collection }) => {
  const mutation = transaction.mutations[0]
  try {
    await api.updateTodo(mutation.original.id, mutation.changes)
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Update failed, rolling back:', error)
    throw error
  }
}

```[schema?](#schema)ts

```
optional schema: TSchema;

```

```
optional schema: TSchema;

```

Defined in:[packages/db/src/types.ts:359](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L359)
[startSync?](#startsync)ts

```
optional startSync: boolean;

```

```
optional startSync: boolean;

```

Defined in:[packages/db/src/types.ts:379](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L379)

Whether to start syncing immediately when the collection is created.
Defaults to false for lazy loading. Set to true to immediately sync.
[sync](#sync)ts

```
sync: SyncConfig<T, TKey>;

```

```
sync: SyncConfig<T, TKey>;

```

Defined in:[packages/db/src/types.ts:358](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L358)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/collectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>