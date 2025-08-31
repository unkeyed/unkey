# Query Collection | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageQuery CollectionOverviewInstallationBasic UsageConfiguration OptionsRequired OptionsQuery OptionsCollection OptionsPersistence HandlersPersistence HandlersControlling Refetch BehaviorUtility MethodsDirect WritesUnderstanding the Data StoresWhen to Use Direct WritesIndividual Write OperationsBatch OperationsReal-World Example: WebSocket IntegrationExample: Incremental UpdatesExample: Large Dataset PaginationImportant BehaviorsFull State SyncEmpty Array BehaviorHandling Partial/Incremental FetchesDirect Writes and Query SyncComplete Direct Write API Reference# Query Collection

Copy Markdown[Query Collection](#query-collection)

Query collections provide seamless integration between TanStack DB and TanStack Query, enabling automatic synchronization between your local database and remote data sources.
[Overview](#overview)

The@tanstack/query-db-collectionpackage allows you to create collections that:

- Automatically sync with remote data via TanStack Query
- Support optimistic updates with automatic rollback on errors
- Handle persistence through customizable mutation handlers
- Provide direct write capabilities for directly writing to the sync store
[Installation](#installation)bash

```
npm install @tanstack/query-db-collection @tanstack/query-core @tanstack/db

```

```
npm install @tanstack/query-db-collection @tanstack/query-core @tanstack/db

```[Basic Usage](#basic-usage)typescript

```
import { QueryClient } from '@tanstack/query-core'
import { createCollection } from '@tanstack/db'
import { queryCollectionOptions } from '@tanstack/query-db-collection'

const queryClient = new QueryClient()

const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    queryClient,
    getKey: (item) => item.id,
  })
)

```

```
import { QueryClient } from '@tanstack/query-core'
import { createCollection } from '@tanstack/db'
import { queryCollectionOptions } from '@tanstack/query-db-collection'

const queryClient = new QueryClient()

const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    queryClient,
    getKey: (item) => item.id,
  })
)

```[Configuration Options](#configuration-options)

ThequeryCollectionOptionsfunction accepts the following options:
[Required Options](#required-options)

- queryKey: The query key for TanStack Query
- queryFn: Function that fetches data from the server
- queryClient: TanStack Query client instance
- getKey: Function to extract the unique key from an item
[Query Options](#query-options)

- enabled: Whether the query should automatically run (default:true)
- refetchInterval: Refetch interval in milliseconds
- retry: Retry configuration for failed queries
- retryDelay: Delay between retries
- staleTime: How long data is considered fresh
- meta: Optional metadata that will be passed to the query function context
[Collection Options](#collection-options)

- id: Unique identifier for the collection
- schema: Schema for validating items
- sync: Custom sync configuration
- startSync: Whether to start syncing immediately (default:true)
[Persistence Handlers](#persistence-handlers)

- onInsert: Handler called before insert operations
- onUpdate: Handler called before update operations
- onDelete: Handler called before delete operations
[Persistence Handlers](#persistence-handlers-1)

You can define handlers that are called when mutations occur. These handlers can persist changes to your backend and control whether the query should refetch after the operation:
typescript

```
const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: fetchTodos,
    queryClient,
    getKey: (item) => item.id,
    
    onInsert: async ({ transaction }) => {
      const newItems = transaction.mutations.map(m => m.modified)
      await api.createTodos(newItems)
      // Returning nothing or { refetch: true } will trigger a refetch
      // Return { refetch: false } to skip automatic refetch
    },
    
    onUpdate: async ({ transaction }) => {
      const updates = transaction.mutations.map(m => ({
        id: m.key,
        changes: m.changes
      }))
      await api.updateTodos(updates)
    },
    
    onDelete: async ({ transaction }) => {
      const ids = transaction.mutations.map(m => m.key)
      await api.deleteTodos(ids)
    }
  })
)

```

```
const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: fetchTodos,
    queryClient,
    getKey: (item) => item.id,
    
    onInsert: async ({ transaction }) => {
      const newItems = transaction.mutations.map(m => m.modified)
      await api.createTodos(newItems)
      // Returning nothing or { refetch: true } will trigger a refetch
      // Return { refetch: false } to skip automatic refetch
    },
    
    onUpdate: async ({ transaction }) => {
      const updates = transaction.mutations.map(m => ({
        id: m.key,
        changes: m.changes
      }))
      await api.updateTodos(updates)
    },
    
    onDelete: async ({ transaction }) => {
      const ids = transaction.mutations.map(m => m.key)
      await api.deleteTodos(ids)
    }
  })
)

```[Controlling Refetch Behavior](#controlling-refetch-behavior)

By default, after any persistence handler (onInsert,onUpdate, oronDelete) completes successfully, the query will automatically refetch to ensure the local state matches the server state.

You can control this behavior by returning an object with arefetchproperty:
typescript

```
onInsert: async ({ transaction }) => {
  await api.createTodos(transaction.mutations.map(m => m.modified))
  
  // Skip the automatic refetch
  return { refetch: false }
}

```

```
onInsert: async ({ transaction }) => {
  await api.createTodos(transaction.mutations.map(m => m.modified))
  
  // Skip the automatic refetch
  return { refetch: false }
}

```

This is useful when:

- You're confident the server state matches what you sent
- You want to avoid unnecessary network requests
- You're handling state updates through other mechanisms (like WebSockets)
[Utility Methods](#utility-methods)

The collection provides these utility methods viacollection.utils:

- refetch(): Manually trigger a refetch of the query
[Direct Writes](#direct-writes)

Direct writes are intended for scenarios where the normal query/mutation flow doesn't fit your needs. They allow you to write directly to the synced data store, bypassing the optimistic update system and query refetch mechanism.
[Understanding the Data Stores](#understanding-the-data-stores)

Query Collections maintain two data stores:

1. **Synced Data Store**- The authoritative state synchronized with the server viaqueryFn
2. **Optimistic Mutations Store**- Temporary changes that are applied optimistically before server confirmation

Normal collection operations (insert, update, delete) create optimistic mutations that are:

- Applied immediately to the UI
- Sent to the server via persistence handlers
- Rolled back automatically if the server request fails
- Replaced with server data when the query refetches

Direct writes bypass this system entirely and write directly to the synced data store, making them ideal for handling real-time updates from alternative sources.
[When to Use Direct Writes](#when-to-use-direct-writes)

Direct writes should be used when:

- You need to sync real-time updates from WebSockets or server-sent events
- You're dealing with large datasets where refetching everything is too expensive
- You receive incremental updates or server-computed field updates
- You need to implement complex pagination or partial data loading scenarios
[Individual Write Operations](#individual-write-operations)typescript

```
// Insert a new item directly to the synced data store
todosCollection.utils.writeInsert({ id: '1', text: 'Buy milk', completed: false })

// Update an existing item in the synced data store
todosCollection.utils.writeUpdate({ id: '1', completed: true })

// Delete an item from the synced data store
todosCollection.utils.writeDelete('1')

// Upsert (insert or update) in the synced data store
todosCollection.utils.writeUpsert({ id: '1', text: 'Buy milk', completed: false })

```

```
// Insert a new item directly to the synced data store
todosCollection.utils.writeInsert({ id: '1', text: 'Buy milk', completed: false })

// Update an existing item in the synced data store
todosCollection.utils.writeUpdate({ id: '1', completed: true })

// Delete an item from the synced data store
todosCollection.utils.writeDelete('1')

// Upsert (insert or update) in the synced data store
todosCollection.utils.writeUpsert({ id: '1', text: 'Buy milk', completed: false })

```

These operations:

- Write directly to the synced data store
- Do NOT create optimistic mutations
- Do NOT trigger automatic query refetches
- Update the TanStack Query cache immediately
- Are immediately visible in the UI
[Batch Operations](#batch-operations)

ThewriteBatchmethod allows you to perform multiple operations atomically. Any write operations called within the callback will be collected and executed as a single transaction:
typescript

```
todosCollection.utils.writeBatch(() => {
  todosCollection.utils.writeInsert({ id: '1', text: 'Buy milk' })
  todosCollection.utils.writeInsert({ id: '2', text: 'Walk dog' })
  todosCollection.utils.writeUpdate({ id: '3', completed: true })
  todosCollection.utils.writeDelete('4')
})

```

```
todosCollection.utils.writeBatch(() => {
  todosCollection.utils.writeInsert({ id: '1', text: 'Buy milk' })
  todosCollection.utils.writeInsert({ id: '2', text: 'Walk dog' })
  todosCollection.utils.writeUpdate({ id: '3', completed: true })
  todosCollection.utils.writeDelete('4')
})

```[Real-World Example: WebSocket Integration](#real-world-example-websocket-integration)typescript

```
// Handle real-time updates from WebSocket without triggering full refetches
ws.on('todos:update', (changes) => {
  todosCollection.utils.writeBatch(() => {
    changes.forEach(change => {
      switch (change.type) {
        case 'insert':
          todosCollection.utils.writeInsert(change.data)
          break
        case 'update':
          todosCollection.utils.writeUpdate(change.data)
          break
        case 'delete':
          todosCollection.utils.writeDelete(change.id)
          break
      }
    })
  })
})

```

```
// Handle real-time updates from WebSocket without triggering full refetches
ws.on('todos:update', (changes) => {
  todosCollection.utils.writeBatch(() => {
    changes.forEach(change => {
      switch (change.type) {
        case 'insert':
          todosCollection.utils.writeInsert(change.data)
          break
        case 'update':
          todosCollection.utils.writeUpdate(change.data)
          break
        case 'delete':
          todosCollection.utils.writeDelete(change.id)
          break
      }
    })
  })
})

```[Example: Incremental Updates](#example-incremental-updates)typescript

```
// Handle server responses after mutations without full refetch
const createTodo = async (todo) => {
  // Optimistically add the todo
  const tempId = crypto.randomUUID()
  todosCollection.insert({ ...todo, id: tempId })
  
  try {
    // Send to server
    const serverTodo = await api.createTodo(todo)
    
    // Sync the server response (with server-generated ID and timestamps)
    // without triggering a full collection refetch
    todosCollection.utils.writeBatch(() => {
      todosCollection.utils.writeDelete(tempId)
      todosCollection.utils.writeInsert(serverTodo)
    })
  } catch (error) {
    // Rollback happens automatically
    throw error
  }
}

```

```
// Handle server responses after mutations without full refetch
const createTodo = async (todo) => {
  // Optimistically add the todo
  const tempId = crypto.randomUUID()
  todosCollection.insert({ ...todo, id: tempId })
  
  try {
    // Send to server
    const serverTodo = await api.createTodo(todo)
    
    // Sync the server response (with server-generated ID and timestamps)
    // without triggering a full collection refetch
    todosCollection.utils.writeBatch(() => {
      todosCollection.utils.writeDelete(tempId)
      todosCollection.utils.writeInsert(serverTodo)
    })
  } catch (error) {
    // Rollback happens automatically
    throw error
  }
}

```[Example: Large Dataset Pagination](#example-large-dataset-pagination)typescript

```
// Load additional pages without refetching existing data
const loadMoreTodos = async (page) => {
  const newTodos = await api.getTodos({ page, limit: 50 })
  
  // Add new items without affecting existing ones
  todosCollection.utils.writeBatch(() => {
    newTodos.forEach(todo => {
      todosCollection.utils.writeInsert(todo)
    })
  })
}

```

```
// Load additional pages without refetching existing data
const loadMoreTodos = async (page) => {
  const newTodos = await api.getTodos({ page, limit: 50 })
  
  // Add new items without affecting existing ones
  todosCollection.utils.writeBatch(() => {
    newTodos.forEach(todo => {
      todosCollection.utils.writeInsert(todo)
    })
  })
}

```[Important Behaviors](#important-behaviors)[Full State Sync](#full-state-sync)

The query collection treats thequeryFnresult as the**complete state**of the collection. This means:

- Items present in the collection but not in the query result will be deleted
- Items in the query result but not in the collection will be inserted
- Items present in both will be updated if they differ
[Empty Array Behavior](#empty-array-behavior)

WhenqueryFnreturns an empty array,**all items in the collection will be deleted**. This is because the collection interprets an empty array as "the server has no items".
typescript

```
// This will delete all items in the collection
queryFn: async () => []

```

```
// This will delete all items in the collection
queryFn: async () => []

```[Handling Partial/Incremental Fetches](#handling-partialincremental-fetches)

Since the query collection expectsqueryFnto return the complete state, you can handle partial fetches by merging new data with existing data:
typescript

```
const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async ({ queryKey }) => {
      // Get existing data from cache
      const existingData = queryClient.getQueryData(queryKey) || []
      
      // Fetch only new/updated items (e.g., changes since last sync)
      const lastSyncTime = localStorage.getItem('todos-last-sync')
      const newData = await fetch(`/api/todos?since=${lastSyncTime}`).then(r => r.json())
      
      // Merge new data with existing data
      const existingMap = new Map(existingData.map(item => [item.id, item]))
      
      // Apply updates and additions
      newData.forEach(item => {
        existingMap.set(item.id, item)
      })
      
      // Handle deletions if your API provides them
      if (newData.deletions) {
        newData.deletions.forEach(id => existingMap.delete(id))
      }
      
      // Update sync time
      localStorage.setItem('todos-last-sync', new Date().toISOString())
      
      // Return the complete merged state
      return Array.from(existingMap.values())
    },
    queryClient,
    getKey: (item) => item.id,
  })
)

```

```
const todosCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async ({ queryKey }) => {
      // Get existing data from cache
      const existingData = queryClient.getQueryData(queryKey) || []
      
      // Fetch only new/updated items (e.g., changes since last sync)
      const lastSyncTime = localStorage.getItem('todos-last-sync')
      const newData = await fetch(`/api/todos?since=${lastSyncTime}`).then(r => r.json())
      
      // Merge new data with existing data
      const existingMap = new Map(existingData.map(item => [item.id, item]))
      
      // Apply updates and additions
      newData.forEach(item => {
        existingMap.set(item.id, item)
      })
      
      // Handle deletions if your API provides them
      if (newData.deletions) {
        newData.deletions.forEach(id => existingMap.delete(id))
      }
      
      // Update sync time
      localStorage.setItem('todos-last-sync', new Date().toISOString())
      
      // Return the complete merged state
      return Array.from(existingMap.values())
    },
    queryClient,
    getKey: (item) => item.id,
  })
)

```

This pattern allows you to:

- Fetch only incremental changes from your API
- Merge those changes with existing data
- Return the complete state that the collection expects
- Avoid the performance overhead of fetching all data every time
[Direct Writes and Query Sync](#direct-writes-and-query-sync)

Direct writes update the collection immediately and also update the TanStack Query cache. However, they do not prevent the normal query sync behavior. If yourqueryFnreturns data that conflicts with your direct writes, the query data will take precedence.

To handle this properly:

1. Use{ refetch: false }in your persistence handlers when using direct writes
2. Set appropriatestaleTimeto prevent unnecessary refetches
3. Design yourqueryFnto be aware of incremental updates (e.g., only fetch new data)
[Complete Direct Write API Reference](#complete-direct-write-api-reference)

All direct write methods are available oncollection.utils:

- writeInsert(data): Insert one or more items directly
- writeUpdate(data): Update one or more items directly
- writeDelete(keys): Delete one or more items directly
- writeUpsert(data): Insert or update one or more items directly
- writeBatch(callback): Perform multiple operations atomically
- refetch(): Manually trigger a refetch of the query[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/collections/query-collection.md)[Creating Collection Options Creators](/db/latest/docs/guides/collection-options-creator)[Electric Collection](/db/latest/docs/collections/electric-collection)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>