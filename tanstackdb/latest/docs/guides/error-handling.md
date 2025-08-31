# Error Handling | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageError HandlingError TypesSchemaValidationErrorCollection Status and Error StatesTransaction Error HandlingTransaction States and Error InformationCollection Operation ErrorsInvalid Collection StateMissing Mutation HandlersDuplicate Key ErrorsSchema Validation ErrorsSync Error HandlingQuery Collection Sync ErrorsSync Write ErrorsCleanup Error HandlingError Recovery PatternsCollection Cleanup and RestartGraceful DegradationTransaction Rollback CascadingHandling Invalid State ErrorsBest PracticesExample: Complete Error HandlingSee Also# Error Handling

Copy Markdown[Error Handling](#error-handling)

TanStack DB provides comprehensive error handling capabilities to ensure robust data synchronization and state management. This guide covers the built-in error handling mechanisms and how to work with them effectively.
[Error Types](#error-types)

TanStack DB provides named error classes for better error handling and type safety. All error classes can be imported from@tanstack/db(or more commonly, the framework-specific package e.g.@tanstack/react-db):
ts

```
import {
  SchemaValidationError,
  CollectionInErrorStateError,
  DuplicateKeyError,
  MissingHandlerError,
  TransactionError,
  // ... and many more
} from "@tanstack/db"

```

```
import {
  SchemaValidationError,
  CollectionInErrorStateError,
  DuplicateKeyError,
  MissingHandlerError,
  TransactionError,
  // ... and many more
} from "@tanstack/db"

```[SchemaValidationError](#schemavalidationerror)

Thrown when data doesn't match the collection's schema during insert or update operations:
ts

```
import { SchemaValidationError } from "@tanstack/db"

try {
  todoCollection.insert({ text: 123 }) // Invalid type
} catch (error) {
  if (error instanceof SchemaValidationError) {
    console.log(error.type) // 'insert' or 'update'
    console.log(error.issues) // Array of validation issues
    // Example issue: { message: "Expected string, received number", path: ["text"] }
  }
}

```

```
import { SchemaValidationError } from "@tanstack/db"

try {
  todoCollection.insert({ text: 123 }) // Invalid type
} catch (error) {
  if (error instanceof SchemaValidationError) {
    console.log(error.type) // 'insert' or 'update'
    console.log(error.issues) // Array of validation issues
    // Example issue: { message: "Expected string, received number", path: ["text"] }
  }
}

```

The error includes:

- type: Whether it was an 'insert' or 'update' operation
- issues: Array of validation issues with messages and paths
- message: A formatted error message listing all issues
[Collection Status and Error States](#collection-status-and-error-states)

Collections track their status and transition between states:
tsx

```
import { useLiveQuery } from "@tanstack/react-db"

const TodoList = () => {
  const { data, status, isError, isLoading, isReady } = useLiveQuery(
    (query) => query.from({ todos: todoCollection })
  )

  if (isError) {
    return <div>Collection is in error state</div>
  }

  if (isLoading) {
    return <div>Loading...</div>
  }

  return <div>{data?.map(todo => <div key={todo.id}>{todo.text}</div>)}</div>
}

```

```
import { useLiveQuery } from "@tanstack/react-db"

const TodoList = () => {
  const { data, status, isError, isLoading, isReady } = useLiveQuery(
    (query) => query.from({ todos: todoCollection })
  )

  if (isError) {
    return <div>Collection is in error state</div>
  }

  if (isLoading) {
    return <div>Loading...</div>
  }

  return <div>{data?.map(todo => <div key={todo.id}>{todo.text}</div>)}</div>
}

```

Collection status values:

- idle- Not yet started
- loading- Loading initial data
- initialCommit- Processing initial data
- ready- Ready for use
- error- In error state
- cleaned-up- Cleaned up and no longer usable
[Transaction Error Handling](#transaction-error-handling)

When mutations fail, TanStack DB automatically rolls back optimistic updates:
ts

```
const todoCollection = createCollection({
  id: "todos",
  onInsert: async ({ transaction }) => {
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify(transaction.mutations[0].modified),
    })
    
    if (!response.ok) {
      // Throwing an error will rollback the optimistic state
      throw new Error(`HTTP Error: ${response.status}`)
    }
    
    return response.json()
  },
})

// Usage - optimistic update will be rolled back if the mutation fails
try {
  const tx = await todoCollection.insert({
    id: "1",
    text: "New todo",
    completed: false,
  })
  
  await tx.isPersisted.promise
} catch (error) {
  // The optimistic update has been automatically rolled back
  console.error("Failed to create todo:", error)
}

```

```
const todoCollection = createCollection({
  id: "todos",
  onInsert: async ({ transaction }) => {
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify(transaction.mutations[0].modified),
    })
    
    if (!response.ok) {
      // Throwing an error will rollback the optimistic state
      throw new Error(`HTTP Error: ${response.status}`)
    }
    
    return response.json()
  },
})

// Usage - optimistic update will be rolled back if the mutation fails
try {
  const tx = await todoCollection.insert({
    id: "1",
    text: "New todo",
    completed: false,
  })
  
  await tx.isPersisted.promise
} catch (error) {
  // The optimistic update has been automatically rolled back
  console.error("Failed to create todo:", error)
}

```[Transaction States and Error Information](#transaction-states-and-error-information)

Transactions have the following states:

- pending- Transaction is being processed
- persisting- Currently executing the mutation function
- completed- Transaction completed successfully
- failed- Transaction failed and was rolled back

Access transaction error information from collection operations:
ts

```
const todoCollection = createCollection({
  id: "todos",
  onUpdate: async ({ transaction }) => {
    const response = await fetch(`/api/todos/${transaction.mutations[0].key}`, {
      method: "PUT",
      body: JSON.stringify(transaction.mutations[0].modified),
    })
    
    if (!response.ok) {
      throw new Error(`Update failed: ${response.status}`)
    }
  },
})

try {
  const tx = await todoCollection.update("todo-1", (draft) => {
    draft.completed = true
  })
  
  await tx.isPersisted.promise
} catch (error) {
  // Transaction has been rolled back
  console.log(tx.state) // "failed"
  console.log(tx.error) // { message: "Update failed: 500", error: Error }
}

```

```
const todoCollection = createCollection({
  id: "todos",
  onUpdate: async ({ transaction }) => {
    const response = await fetch(`/api/todos/${transaction.mutations[0].key}`, {
      method: "PUT",
      body: JSON.stringify(transaction.mutations[0].modified),
    })
    
    if (!response.ok) {
      throw new Error(`Update failed: ${response.status}`)
    }
  },
})

try {
  const tx = await todoCollection.update("todo-1", (draft) => {
    draft.completed = true
  })
  
  await tx.isPersisted.promise
} catch (error) {
  // Transaction has been rolled back
  console.log(tx.state) // "failed"
  console.log(tx.error) // { message: "Update failed: 500", error: Error }
}

```

Or with manual transaction creation:
ts

```
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    throw new Error("API failed")
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

try {
  await tx.commit()
} catch (error) {
  // Transaction has been rolled back
  console.log(tx.state) // "failed"
  console.log(tx.error) // { message: "API failed", error: Error }
}

```

```
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    throw new Error("API failed")
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

try {
  await tx.commit()
} catch (error) {
  // Transaction has been rolled back
  console.log(tx.state) // "failed"
  console.log(tx.error) // { message: "API failed", error: Error }
}

```[Collection Operation Errors](#collection-operation-errors)[Invalid Collection State](#invalid-collection-state)

Collections in anerrorstate cannot perform operations and must be manually recovered:
ts

```
import { CollectionInErrorStateError } from "@tanstack/db"

try {
  todoCollection.insert(newTodo)
} catch (error) {
  if (error instanceof CollectionInErrorStateError) {
    // Collection needs to be cleaned up and restarted
    await todoCollection.cleanup()
    
    // Now retry the operation
    todoCollection.insert(newTodo)
  }
}

```

```
import { CollectionInErrorStateError } from "@tanstack/db"

try {
  todoCollection.insert(newTodo)
} catch (error) {
  if (error instanceof CollectionInErrorStateError) {
    // Collection needs to be cleaned up and restarted
    await todoCollection.cleanup()
    
    // Now retry the operation
    todoCollection.insert(newTodo)
  }
}

```[Missing Mutation Handlers](#missing-mutation-handlers)

Direct mutations require handlers to be configured:
ts

```
const todoCollection = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  // Missing onInsert handler
})

// This will throw an error
todoCollection.insert(newTodo)
// Error: Collection.insert called directly (not within an explicit transaction) but no 'onInsert' handler is configured

```

```
const todoCollection = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  // Missing onInsert handler
})

// This will throw an error
todoCollection.insert(newTodo)
// Error: Collection.insert called directly (not within an explicit transaction) but no 'onInsert' handler is configured

```[Duplicate Key Errors](#duplicate-key-errors)

Inserting items with existing keys will throw:
ts

```
import { DuplicateKeyError } from "@tanstack/db"

try {
  todoCollection.insert({ id: "existing-id", text: "Todo" })
} catch (error) {
  if (error instanceof DuplicateKeyError) {
    console.log(`Duplicate key: ${error.message}`)
  }
}

```

```
import { DuplicateKeyError } from "@tanstack/db"

try {
  todoCollection.insert({ id: "existing-id", text: "Todo" })
} catch (error) {
  if (error instanceof DuplicateKeyError) {
    console.log(`Duplicate key: ${error.message}`)
  }
}

```[Schema Validation Errors](#schema-validation-errors)

Schema validation must be synchronous:
ts

```
const todoCollection = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  schema: {
    "~standard": {
      validate: async (data) => { // Async validation not allowed
        // ...
      }
    }
  }
})

// Will throw: Schema validation must be synchronous

```

```
const todoCollection = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  schema: {
    "~standard": {
      validate: async (data) => { // Async validation not allowed
        // ...
      }
    }
  }
})

// Will throw: Schema validation must be synchronous

```[Sync Error Handling](#sync-error-handling)[Query Collection Sync Errors](#query-collection-sync-errors)

Query collections handle sync errors gracefully and mark the collection as ready even on error to avoid blocking applications:
ts

```
import { queryCollectionOptions } from "@tanstack/query-db-collection"

const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ["todos"],
    queryFn: async () => {
      const response = await fetch("/api/todos")
      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.status}`)
      }
      return response.json()
    },
    queryClient,
    getKey: (item) => item.id,
    schema: todoSchema,
    // Standard TanStack Query error handling options
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  })
)

```

```
import { queryCollectionOptions } from "@tanstack/query-db-collection"

const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ["todos"],
    queryFn: async () => {
      const response = await fetch("/api/todos")
      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.status}`)
      }
      return response.json()
    },
    queryClient,
    getKey: (item) => item.id,
    schema: todoSchema,
    // Standard TanStack Query error handling options
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  })
)

```

When sync errors occur:

- Error is logged to console:[QueryCollection] Error observing query...
- Collection is marked as ready to prevent blocking the application
- Cached data remains available
[Sync Write Errors](#sync-write-errors)

Sync functions must handle their own errors during write operations:
ts

```
const collection = createCollection({
  id: "todos",
  sync: {
    sync: ({ begin, write, commit }) => {
      begin()
      
      try {
        // Will throw if key already exists
        write({ type: "insert", value: { id: "existing-id", text: "Todo" } })
      } catch (error) {
        // Error: Cannot insert document with key "existing-id" from sync because it already exists
      }
      
      commit()
    }
  }
})

```

```
const collection = createCollection({
  id: "todos",
  sync: {
    sync: ({ begin, write, commit }) => {
      begin()
      
      try {
        // Will throw if key already exists
        write({ type: "insert", value: { id: "existing-id", text: "Todo" } })
      } catch (error) {
        // Error: Cannot insert document with key "existing-id" from sync because it already exists
      }
      
      commit()
    }
  }
})

```[Cleanup Error Handling](#cleanup-error-handling)

Cleanup errors are isolated to prevent blocking the cleanup process:
ts

```
const collection = createCollection({
  id: "todos",
  sync: {
    sync: ({ begin, commit }) => {
      begin()
      commit()
      
      // Return a cleanup function
      return () => {
        // If this throws, the error is re-thrown in a microtask
        // but cleanup continues successfully
        throw new Error("Sync cleanup failed")
      }
    },
  },
})

// Cleanup completes even if the sync cleanup function throws
await collection.cleanup() // Resolves successfully
// Error is re-thrown asynchronously via queueMicrotask

```

```
const collection = createCollection({
  id: "todos",
  sync: {
    sync: ({ begin, commit }) => {
      begin()
      commit()
      
      // Return a cleanup function
      return () => {
        // If this throws, the error is re-thrown in a microtask
        // but cleanup continues successfully
        throw new Error("Sync cleanup failed")
      }
    },
  },
})

// Cleanup completes even if the sync cleanup function throws
await collection.cleanup() // Resolves successfully
// Error is re-thrown asynchronously via queueMicrotask

```[Error Recovery Patterns](#error-recovery-patterns)[Collection Cleanup and Restart](#collection-cleanup-and-restart)

Clean up collections in error states:
ts

```
if (todoCollection.status === "error") {
  // Cleanup will stop sync and reset the collection
  await todoCollection.cleanup()
  
  // Collection will automatically restart on next access
  todoCollection.preload() // Or any other operation
}

```

```
if (todoCollection.status === "error") {
  // Cleanup will stop sync and reset the collection
  await todoCollection.cleanup()
  
  // Collection will automatically restart on next access
  todoCollection.preload() // Or any other operation
}

```[Graceful Degradation](#graceful-degradation)

Collections continue to work with cached data even when sync fails:
tsx

```
const TodoApp = () => {
  const { data, isError } = useLiveQuery((query) => 
    query.from({ todos: todoCollection })
  )

  return (
    <div>
      {isError && (
        <div>Sync failed, but you can still view cached data</div>
      )}
      {data?.map(todo => <TodoItem key={todo.id} todo={todo} />)}
    </div>
  )
}

```

```
const TodoApp = () => {
  const { data, isError } = useLiveQuery((query) => 
    query.from({ todos: todoCollection })
  )

  return (
    <div>
      {isError && (
        <div>Sync failed, but you can still view cached data</div>
      )}
      {data?.map(todo => <TodoItem key={todo.id} todo={todo} />)}
    </div>
  )
}

```[Transaction Rollback Cascading](#transaction-rollback-cascading)

When a transaction fails, conflicting transactions are automatically rolled back:
ts

```
const tx1 = createTransaction({ mutationFn: async () => {} })
const tx2 = createTransaction({ mutationFn: async () => {} })

tx1.mutate(() => collection.update("1", draft => { draft.value = "A" }))
tx2.mutate(() => collection.update("1", draft => { draft.value = "B" })) // Same item

// Rolling back tx1 will also rollback tx2 due to conflict
tx1.rollback() // tx2 is automatically rolled back

```

```
const tx1 = createTransaction({ mutationFn: async () => {} })
const tx2 = createTransaction({ mutationFn: async () => {} })

tx1.mutate(() => collection.update("1", draft => { draft.value = "A" }))
tx2.mutate(() => collection.update("1", draft => { draft.value = "B" })) // Same item

// Rolling back tx1 will also rollback tx2 due to conflict
tx1.rollback() // tx2 is automatically rolled back

```[Handling Invalid State Errors](#handling-invalid-state-errors)

Transactions validate their state before operations:
ts

```
const tx = createTransaction({ mutationFn: async () => {} })

// Complete the transaction
await tx.commit()

// These will throw:
tx.mutate(() => {}) // Error: You can no longer call .mutate() as the transaction is no longer pending
tx.commit() // Error: You can no longer call .commit() as the transaction is no longer pending
tx.rollback() // Error: You can no longer call .rollback() as the transaction is already completed

```

```
const tx = createTransaction({ mutationFn: async () => {} })

// Complete the transaction
await tx.commit()

// These will throw:
tx.mutate(() => {}) // Error: You can no longer call .mutate() as the transaction is no longer pending
tx.commit() // Error: You can no longer call .commit() as the transaction is no longer pending
tx.rollback() // Error: You can no longer call .rollback() as the transaction is already completed

```[Best Practices](#best-practices)

1. **Use instanceof checks**- Useinstanceofinstead of string matching for error handling:
ts

```
// ✅ Good - type-safe error handling
if (error instanceof SchemaValidationError) {
  // Handle validation error
}

// ❌ Avoid - brittle string matching  
if (error.message.includes("validation failed")) {
  // Handle validation error
}

```

```
// ✅ Good - type-safe error handling
if (error instanceof SchemaValidationError) {
  // Handle validation error
}

// ❌ Avoid - brittle string matching  
if (error.message.includes("validation failed")) {
  // Handle validation error
}

```
2. **Import specific error types**- Import only the error classes you need for better tree-shaking
3. **Always handle SchemaValidationError**- Provide clear feedback for validation failures
4. **Check collection status**- UseisError,isLoading,isReadyflags in React components
5. **Handle transaction promises**- Always handleisPersisted.promiserejections
[Example: Complete Error Handling](#example-complete-error-handling)tsx

```
import { 
  createCollection, 
  SchemaValidationError,
  DuplicateKeyError,
  createTransaction
} from "@tanstack/db"
import { useLiveQuery } from "@tanstack/react-db"

const todoCollection = createCollection({
  id: "todos",
  schema: todoSchema,
  getKey: (todo) => todo.id,
  onInsert: async ({ transaction }) => {
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify(transaction.mutations[0].modified),
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    return response.json()
  },
  sync: {
    sync: ({ begin, write, commit }) => {
      // Your sync implementation
      begin()
      // ... sync logic
      commit()
    }
  }
})

const TodoApp = () => {
  const { data, status, isError, isLoading } = useLiveQuery(
    (query) => query.from({ todos: todoCollection })
  )

  const handleAddTodo = async (text: string) => {
    try {
      const tx = await todoCollection.insert({
        id: crypto.randomUUID(),
        text,
        completed: false,
      })
      
      // Wait for persistence
      await tx.isPersisted.promise
    } catch (error) {
      if (error instanceof SchemaValidationError) {
        alert(`Validation error: ${error.issues[0]?.message}`)
      } else if (error instanceof DuplicateKeyError) {
        alert("A todo with this ID already exists")
      } else {
        alert(`Failed to add todo: ${error.message}`)
      }
    }
  }

  const handleCleanup = async () => {
    try {
      await todoCollection.cleanup()
      // Collection will restart on next access
    } catch (error) {
      console.error("Cleanup failed:", error)
    }
  }

  if (isError) {
    return (
      <div>
        <div>Collection error - data may be stale</div>
        <button onClick={handleCleanup}>
          Restart Collection
        </button>
      </div>
    )
  }

  if (isLoading) {
    return <div>Loading todos...</div>
  }

  return (
    <div>
      <button onClick={() => handleAddTodo("New todo")}>
        Add Todo
      </button>
      {data?.map(todo => (
        <div key={todo.id}>{todo.text}</div>
      ))}
    </div>
  )
}

```

```
import { 
  createCollection, 
  SchemaValidationError,
  DuplicateKeyError,
  createTransaction
} from "@tanstack/db"
import { useLiveQuery } from "@tanstack/react-db"

const todoCollection = createCollection({
  id: "todos",
  schema: todoSchema,
  getKey: (todo) => todo.id,
  onInsert: async ({ transaction }) => {
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify(transaction.mutations[0].modified),
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    return response.json()
  },
  sync: {
    sync: ({ begin, write, commit }) => {
      // Your sync implementation
      begin()
      // ... sync logic
      commit()
    }
  }
})

const TodoApp = () => {
  const { data, status, isError, isLoading } = useLiveQuery(
    (query) => query.from({ todos: todoCollection })
  )

  const handleAddTodo = async (text: string) => {
    try {
      const tx = await todoCollection.insert({
        id: crypto.randomUUID(),
        text,
        completed: false,
      })
      
      // Wait for persistence
      await tx.isPersisted.promise
    } catch (error) {
      if (error instanceof SchemaValidationError) {
        alert(`Validation error: ${error.issues[0]?.message}`)
      } else if (error instanceof DuplicateKeyError) {
        alert("A todo with this ID already exists")
      } else {
        alert(`Failed to add todo: ${error.message}`)
      }
    }
  }

  const handleCleanup = async () => {
    try {
      await todoCollection.cleanup()
      // Collection will restart on next access
    } catch (error) {
      console.error("Cleanup failed:", error)
    }
  }

  if (isError) {
    return (
      <div>
        <div>Collection error - data may be stale</div>
        <button onClick={handleCleanup}>
          Restart Collection
        </button>
      </div>
    )
  }

  if (isLoading) {
    return <div>Loading todos...</div>
  }

  return (
    <div>
      <button onClick={() => handleAddTodo("New todo")}>
        Add Todo
      </button>
      {data?.map(todo => (
        <div key={todo.id}>{todo.text}</div>
      ))}
    </div>
  )
}

```[See Also](#see-also)

- [API Reference](/db/latest/docs/overview#api-reference)- Detailed API documentation
- [Mutations Guide](/db/latest/docs/overview#making-optimistic-mutations)- Learn about optimistic updates and rollbacks
- [TanStack Query Error Handling](https://tanstack.com/query/latest/docs/react/guides/error-handling)- Query-specific error handling[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/guides/error-handling.md)[Live Queries](/db/latest/docs/guides/live-queries)[Creating Collection Options Creators](/db/latest/docs/guides/collection-options-creator)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>