# Quick Start | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInstallation1. Create a Collection2. Query with Live Queries3. Optimistic MutationsNext Steps# Quick Start

Copy MarkdownTanStack DB is a reactive client store for building super fast apps. This example will show you how to:

- **Load data**into collections using TanStack Query
- **Query data**with blazing fast live queries
- **Mutate data**with instant optimistic updates
tsx

```
import { createCollection, eq, useLiveQuery } from '@tanstack/react-db'
import { queryCollectionOptions } from '@tanstack/query-db-collection'

// Define a collection that loads data using TanStack Query
const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    getKey: (item) => item.id,
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, {
        method: 'PUT',
        body: JSON.stringify(modified),
      })
    },
  })
)

function Todos() {
  // Live query that updates automatically when data changes
  const { data: todos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .where(({ todo }) => eq(todo.completed, false))
     .orderBy(({ todo }) => todo.createdAt, 'desc')
  )

  const toggleTodo = (todo) => {
    // Instantly applies optimistic state, then syncs to server
    todoCollection.update(todo.id, (draft) => {
      draft.completed = !draft.completed
    })
  }

  return (
    <ul>
      {todos.map((todo) => (
        <li key={todo.id} onClick={() => toggleTodo(todo)}>
          {todo.text}
        </li>
      ))}
    </ul>
  )
}

```

```
import { createCollection, eq, useLiveQuery } from '@tanstack/react-db'
import { queryCollectionOptions } from '@tanstack/query-db-collection'

// Define a collection that loads data using TanStack Query
const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    getKey: (item) => item.id,
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, {
        method: 'PUT',
        body: JSON.stringify(modified),
      })
    },
  })
)

function Todos() {
  // Live query that updates automatically when data changes
  const { data: todos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .where(({ todo }) => eq(todo.completed, false))
     .orderBy(({ todo }) => todo.createdAt, 'desc')
  )

  const toggleTodo = (todo) => {
    // Instantly applies optimistic state, then syncs to server
    todoCollection.update(todo.id, (draft) => {
      draft.completed = !draft.completed
    })
  }

  return (
    <ul>
      {todos.map((todo) => (
        <li key={todo.id} onClick={() => toggleTodo(todo)}>
          {todo.text}
        </li>
      ))}
    </ul>
  )
}

```

You now have collections, live queries, and optimistic mutations! Let's break this down further.
[Installation](#installation)bash

```
npm install @tanstack/react-db @tanstack/query-db-collection

```

```
npm install @tanstack/react-db @tanstack/query-db-collection

```[1. Create a Collection](#1-create-a-collection)

Collections store your data and handle persistence. ThequeryCollectionOptionsloads data using TanStack Query and defines mutation handlers for server sync:
tsx

```
const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    getKey: (item) => item.id,
    // Handle all CRUD operations
    onInsert: async ({ transaction }) => {
      const { modified: newTodo } = transaction.mutations[0]
      await fetch('/api/todos', {
        method: 'POST',
        body: JSON.stringify(newTodo),
      })
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, {
        method: 'PUT', 
        body: JSON.stringify(modified),
      })
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, { method: 'DELETE' })
    },
  })
)

```

```
const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ['todos'],
    queryFn: async () => {
      const response = await fetch('/api/todos')
      return response.json()
    },
    getKey: (item) => item.id,
    // Handle all CRUD operations
    onInsert: async ({ transaction }) => {
      const { modified: newTodo } = transaction.mutations[0]
      await fetch('/api/todos', {
        method: 'POST',
        body: JSON.stringify(newTodo),
      })
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, {
        method: 'PUT', 
        body: JSON.stringify(modified),
      })
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0]
      await fetch(`/api/todos/${original.id}`, { method: 'DELETE' })
    },
  })
)

```[2. Query with Live Queries](#2-query-with-live-queries)

Live queries reactively update when data changes. They support filtering, sorting, joins, and transformations:
tsx

```
function TodoList() {
  // Basic filtering and sorting
  const { data: incompleteTodos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .where(({ todo }) => eq(todo.completed, false))
     .orderBy(({ todo }) => todo.createdAt, 'desc')
  )

  // Transform the data
  const { data: todoSummary } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .select(({ todo }) => ({
       id: todo.id,
       summary: `${todo.text} (${todo.completed ? 'done' : 'pending'})`,
       priority: todo.priority || 'normal'
     }))
  )

  return <div>{/* Render todos */}</div>
}

```

```
function TodoList() {
  // Basic filtering and sorting
  const { data: incompleteTodos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .where(({ todo }) => eq(todo.completed, false))
     .orderBy(({ todo }) => todo.createdAt, 'desc')
  )

  // Transform the data
  const { data: todoSummary } = useLiveQuery((q) =>
    q.from({ todo: todoCollection })
     .select(({ todo }) => ({
       id: todo.id,
       summary: `${todo.text} (${todo.completed ? 'done' : 'pending'})`,
       priority: todo.priority || 'normal'
     }))
  )

  return <div>{/* Render todos */}</div>
}

```[3. Optimistic Mutations](#3-optimistic-mutations)

Mutations apply instantly and sync to your server. If the server request fails, changes automatically roll back:
tsx

```
function TodoActions({ todo }) {
  const addTodo = () => {
    todoCollection.insert({
      id: crypto.randomUUID(),
      text: 'New todo',
      completed: false,
      createdAt: new Date(),
    })
  }

  const toggleComplete = () => {
    todoCollection.update(todo.id, (draft) => {
      draft.completed = !draft.completed
    })
  }

  const updateText = (newText) => {
    todoCollection.update(todo.id, (draft) => {
      draft.text = newText
    })
  }

  const deleteTodo = () => {
    todoCollection.delete(todo.id)
  }

  return (
    <div>
      <button onClick={addTodo}>Add Todo</button>
      <button onClick={toggleComplete}>Toggle</button>
      <button onClick={() => updateText('Updated!')}>Edit</button>
      <button onClick={deleteTodo}>Delete</button>
    </div>
  )
}

```

```
function TodoActions({ todo }) {
  const addTodo = () => {
    todoCollection.insert({
      id: crypto.randomUUID(),
      text: 'New todo',
      completed: false,
      createdAt: new Date(),
    })
  }

  const toggleComplete = () => {
    todoCollection.update(todo.id, (draft) => {
      draft.completed = !draft.completed
    })
  }

  const updateText = (newText) => {
    todoCollection.update(todo.id, (draft) => {
      draft.text = newText
    })
  }

  const deleteTodo = () => {
    todoCollection.delete(todo.id)
  }

  return (
    <div>
      <button onClick={addTodo}>Add Todo</button>
      <button onClick={toggleComplete}>Toggle</button>
      <button onClick={() => updateText('Updated!')}>Edit</button>
      <button onClick={deleteTodo}>Delete</button>
    </div>
  )
}

```[Next Steps](#next-steps)

You now understand the basics of TanStack DB! The collection loads and persists data, live queries provide reactive views, and mutations give instant feedback with automatic server sync.

Explore the docs to learn more about:

- **Installation**- All framework and collection packages
- **Overview**- Complete feature overview and examples
- **Live Queries**- Advanced querying, joins, and aggregations[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/quick-start.md)[Overview](/db/latest/docs/overview)[Installation](/db/latest/docs/installation)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>