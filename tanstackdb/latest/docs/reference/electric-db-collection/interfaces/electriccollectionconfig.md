# ElectricCollectionConfig | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: ElectricCollectionConfig<TExplicit, TSchema, TFallback>RemarksType ParametersPropertiesgetKey()ParametersitemReturnsid?onDelete()?ParametersparamsReturnsExamplesonInsert()?ParametersparamsReturnsExamplesonUpdate()?ParametersparamsReturnsExamplesschema?shapeOptionssync?# ElectricCollectionConfig

Copy Markdown[Interface: ElectricCollectionConfig<TExplicit, TSchema, TFallback>](#interface-electriccollectionconfigtexplicit-tschema-tfallback)

Defined in:[packages/electric-db-collection/src/electric.ts:73](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L73)

Configuration interface for Electric collection options
[Remarks](#remarks)

Type resolution follows a priority order:

1. If you provide an explicit type via generic parameter, it will be used
2. If no explicit type is provided but a schema is, the schema's output type will be inferred
3. If neither explicit type nor schema is provided, the fallback type will be used

You should provide EITHER an explicit type OR a schema, but not both, as they would conflict.
[Type Parameters](#type-parameters)

•**TExplicit***extends*Row<unknown> =Row<unknown>

The explicit type of items in the collection (highest priority)

•**TSchema***extends*StandardSchemaV1=never

The schema type for validation and type inference (second priority)

•**TFallback***extends*Row<unknown> =Row<unknown>

The fallback type if no explicit or schema type is provided
[Properties](#properties)[getKey()](#getkey)ts

```
getKey: (item) => string | number;

```

```
getKey: (item) => string | number;

```

Defined in:[packages/electric-db-collection/src/electric.ts:90](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L90)
[Parameters](#parameters)[item](#item)

ResolveType
[Returns](#returns)

string|number
[id?](#id)ts

```
optional id: string;

```

```
optional id: string;

```

Defined in:[packages/electric-db-collection/src/electric.ts:88](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L88)

All standard Collection configuration properties
[onDelete()?](#ondelete)ts

```
optional onDelete: (params) => Promise<{
  txid: number | number[];
}>;

```

```
optional onDelete: (params) => Promise<{
  txid: number | number[];
}>;

```

Defined in:[packages/electric-db-collection/src/electric.ts:246](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L246)

Optional asynchronous handler function called before a delete operation
Must return an object containing a txid number or array of txids
[Parameters](#parameters-1)[params](#params)

DeleteMutationFnParams<ResolveType<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-1)

Promise<{txid:number|number[];
 }>

Promise resolving to an object with txid or txids
[Examples](#examples)ts

```
// Basic Electric delete handler - MUST return { txid: number }
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  const result = await api.todos.delete({
    id: mutation.original.id
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```

```
// Basic Electric delete handler - MUST return { txid: number }
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  const result = await api.todos.delete({
    id: mutation.original.id
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```ts

```
// Delete handler with multiple items - return array of txids
onDelete: async ({ transaction }) => {
  const deletes = await Promise.all(
    transaction.mutations.map(m =>
      api.todos.delete({
        where: { id: m.key }
      })
    )
  )
  return { txid: deletes.map(d => d.txid) } // Array of txids
}

```

```
// Delete handler with multiple items - return array of txids
onDelete: async ({ transaction }) => {
  const deletes = await Promise.all(
    transaction.mutations.map(m =>
      api.todos.delete({
        where: { id: m.key }
      })
    )
  )
  return { txid: deletes.map(d => d.txid) } // Array of txids
}

```ts

```
// Delete handler with batch operation - single txid
onDelete: async ({ transaction }) => {
  const idsToDelete = transaction.mutations.map(m => m.original.id)
  const result = await api.todos.deleteMany({
    ids: idsToDelete
  })
  return { txid: result.txid } // Single txid for batch operation
}

```

```
// Delete handler with batch operation - single txid
onDelete: async ({ transaction }) => {
  const idsToDelete = transaction.mutations.map(m => m.original.id)
  const result = await api.todos.deleteMany({
    ids: idsToDelete
  })
  return { txid: result.txid } // Single txid for batch operation
}

```ts

```
// Delete handler with optimistic rollback
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  try {
    const result = await api.deleteTodo(mutation.original.id)
    return { txid: result.txid }
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Delete failed, rolling back:', error)
    throw error
  }
}

```

```
// Delete handler with optimistic rollback
onDelete: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  try {
    const result = await api.deleteTodo(mutation.original.id)
    return { txid: result.txid }
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Delete failed, rolling back:', error)
    throw error
  }
}

```[onInsert()?](#oninsert)ts

```
optional onInsert: (params) => Promise<{
  txid: number | number[];
}>;

```

```
optional onInsert: (params) => Promise<{
  txid: number | number[];
}>;

```

Defined in:[packages/electric-db-collection/src/electric.ts:141](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L141)

Optional asynchronous handler function called before an insert operation
Must return an object containing a txid number or array of txids
[Parameters](#parameters-2)[params](#params-1)

InsertMutationFnParams<ResolveType<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-2)

Promise<{txid:number|number[];
 }>

Promise resolving to an object with txid or txids
[Examples](#examples-1)ts

```
// Basic Electric insert handler - MUST return { txid: number }
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  const result = await api.todos.create({
    data: newItem
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```

```
// Basic Electric insert handler - MUST return { txid: number }
onInsert: async ({ transaction }) => {
  const newItem = transaction.mutations[0].modified
  const result = await api.todos.create({
    data: newItem
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```ts

```
// Insert handler with multiple items - return array of txids
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  const results = await Promise.all(
    items.map(item => api.todos.create({ data: item }))
  )
  return { txid: results.map(r => r.txid) } // Array of txids
}

```

```
// Insert handler with multiple items - return array of txids
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  const results = await Promise.all(
    items.map(item => api.todos.create({ data: item }))
  )
  return { txid: results.map(r => r.txid) } // Array of txids
}

```ts

```
// Insert handler with error handling
onInsert: async ({ transaction }) => {
  try {
    const newItem = transaction.mutations[0].modified
    const result = await api.createTodo(newItem)
    return { txid: result.txid }
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // This will cause the transaction to fail
  }
}

```

```
// Insert handler with error handling
onInsert: async ({ transaction }) => {
  try {
    const newItem = transaction.mutations[0].modified
    const result = await api.createTodo(newItem)
    return { txid: result.txid }
  } catch (error) {
    console.error('Insert failed:', error)
    throw error // This will cause the transaction to fail
  }
}

```ts

```
// Insert handler with batch operation - single txid
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  const result = await api.todos.createMany({
    data: items
  })
  return { txid: result.txid } // Single txid for batch operation
}

```

```
// Insert handler with batch operation - single txid
onInsert: async ({ transaction }) => {
  const items = transaction.mutations.map(m => m.modified)
  const result = await api.todos.createMany({
    data: items
  })
  return { txid: result.txid } // Single txid for batch operation
}

```[onUpdate()?](#onupdate)ts

```
optional onUpdate: (params) => Promise<{
  txid: number | number[];
}>;

```

```
optional onUpdate: (params) => Promise<{
  txid: number | number[];
}>;

```

Defined in:[packages/electric-db-collection/src/electric.ts:189](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L189)

Optional asynchronous handler function called before an update operation
Must return an object containing a txid number or array of txids
[Parameters](#parameters-3)[params](#params-2)

UpdateMutationFnParams<ResolveType<TExplicit,TSchema,TFallback>>

Object containing transaction and collection information
[Returns](#returns-3)

Promise<{txid:number|number[];
 }>

Promise resolving to an object with txid or txids
[Examples](#examples-2)ts

```
// Basic Electric update handler - MUST return { txid: number }
onUpdate: async ({ transaction }) => {
  const { original, changes } = transaction.mutations[0]
  const result = await api.todos.update({
    where: { id: original.id },
    data: changes // Only the changed fields
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```

```
// Basic Electric update handler - MUST return { txid: number }
onUpdate: async ({ transaction }) => {
  const { original, changes } = transaction.mutations[0]
  const result = await api.todos.update({
    where: { id: original.id },
    data: changes // Only the changed fields
  })
  return { txid: result.txid } // Required for Electric sync matching
}

```ts

```
// Update handler with multiple items - return array of txids
onUpdate: async ({ transaction }) => {
  const updates = await Promise.all(
    transaction.mutations.map(m =>
      api.todos.update({
        where: { id: m.original.id },
        data: m.changes
      })
    )
  )
  return { txid: updates.map(u => u.txid) } // Array of txids
}

```

```
// Update handler with multiple items - return array of txids
onUpdate: async ({ transaction }) => {
  const updates = await Promise.all(
    transaction.mutations.map(m =>
      api.todos.update({
        where: { id: m.original.id },
        data: m.changes
      })
    )
  )
  return { txid: updates.map(u => u.txid) } // Array of txids
}

```ts

```
// Update handler with optimistic rollback
onUpdate: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  try {
    const result = await api.updateTodo(mutation.original.id, mutation.changes)
    return { txid: result.txid }
  } catch (error) {
    // Transaction will automatically rollback optimistic changes
    console.error('Update failed, rolling back:', error)
    throw error
  }
}

```

```
// Update handler with optimistic rollback
onUpdate: async ({ transaction }) => {
  const mutation = transaction.mutations[0]
  try {
    const result = await api.updateTodo(mutation.original.id, mutation.changes)
    return { txid: result.txid }
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

Defined in:[packages/electric-db-collection/src/electric.ts:89](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L89)
[shapeOptions](#shapeoptions)ts

```
shapeOptions: ShapeStreamOptions<GetExtensions<ResolveType<TExplicit, TSchema, TFallback>>>;

```

```
shapeOptions: ShapeStreamOptions<GetExtensions<ResolveType<TExplicit, TSchema, TFallback>>>;

```

Defined in:[packages/electric-db-collection/src/electric.ts:81](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L81)

Configuration options for the ElectricSQL ShapeStream
[sync?](#sync)ts

```
optional sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```

```
optional sync: SyncConfig<ResolveType<TExplicit, TSchema, TFallback>, string | number>;

```

Defined in:[packages/electric-db-collection/src/electric.ts:91](https://github.com/TanStack/db/blob/main/packages/electric-db-collection/src/electric.ts#L91)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/electric-db-collection/interfaces/electriccollectionconfig.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>