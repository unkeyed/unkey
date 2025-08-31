# Overview | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageTanStack DB - DocumentationContentsHow it worksDefining collectionsUsing live queriesMaking optimistic mutationsExplicit transactionsUni-directional data flowAPI referenceCollectionsCollection schemasQueryCollectionElectricCollectionTrailBaseCollectionLocalStorageCollectionLocalOnlyCollectionDerived collectionsCollectionLive queriesuseLiveQuery hookqueryBuilderTransactional mutatorsmutationFncreateOptimisticActionManual TransactionsTransaction lifecycleWrite operationsinsertupdatedeleteControlling optimistic behaviorWhen to use optimistic: falseBehavior differencesUsage examples1. TanStack Query2. ElectricSQL syncReact NativeMore info# Overview

Copy Markdown[TanStack DB - Documentation](#tanstack-db---documentation)

Welcome to the TanStack DB documentation.

TanStack DB is a reactive client store for building super fast apps on sync. It extends TanStack Query with collections, live queries and optimistic mutations.
[Contents](#contents)

- [How it works](#how-it-works)— understand the TanStack DB development model and how the pieces fit together
- [API reference](#api-reference)— for the primitives and function interfaces
- [Usage examples](#usage-examples)— examples of common usage patterns
- [More info](#more-info)— where to find support and more information
[How it works](#how-it-works)

TanStack DB works by:

- [defining collections](#defining-collections)typed sets of objects that can be populated with data
- [using live queries](#using-live-queries)to query data from/across collections
- [making optimistic mutations](#making-optimistic-mutations)using transactional mutators
tsx

```
// Define collections to load data into
const todoCollection = createCollection({
  // ...your config
  onUpdate: updateMutationFn,
})

const Todos = () => {
  // Bind data using live queries
  const { data: todos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection }).where(({ todo }) => todo.completed)
  )

  const complete = (todo) => {
    // Instantly applies optimistic state
    todoCollection.update(todo.id, (draft) => {
      draft.completed = true
    })
  }

  return (
    <ul>
      {todos.map((todo) => (
        <li key={todo.id} onClick={() => complete(todo)}>
          {todo.text}
        </li>
      ))}
    </ul>
  )
}

```

```
// Define collections to load data into
const todoCollection = createCollection({
  // ...your config
  onUpdate: updateMutationFn,
})

const Todos = () => {
  // Bind data using live queries
  const { data: todos } = useLiveQuery((q) =>
    q.from({ todo: todoCollection }).where(({ todo }) => todo.completed)
  )

  const complete = (todo) => {
    // Instantly applies optimistic state
    todoCollection.update(todo.id, (draft) => {
      draft.completed = true
    })
  }

  return (
    <ul>
      {todos.map((todo) => (
        <li key={todo.id} onClick={() => complete(todo)}>
          {todo.text}
        </li>
      ))}
    </ul>
  )
}

```[Defining collections](#defining-collections)

Collections are typed sets of objects that can be populated with data. They're designed to de-couple loading data into your app from binding data to your components.

Collections can be populated in many ways, including:

- fetching data, for example[from API endpoints using TanStack Query](https://tanstack.com/query/latest)
- syncing data, for example[using a sync engine like ElectricSQL](https://electric-sql.com/)
- storing local data, for example[using localStorage for user preferences and settings](#localstoragecollection)or[in-memory client data or UI state](#localonlycollection)
- from live collection queries, creating[derived collections as materialised views](#using-live-queries)

Once you have your data in collections, you can query across them using live queries in your components.
[Using live queries](#using-live-queries)

Live queries are used to query data out of collections. Live queries are reactive: when the underlying data changes in a way that would affect the query result, the result is incrementally updated and returned from the query, triggering a re-render.

TanStack DB live queries are implemented using[d2ts](https://github.com/electric-sql/d2ts), a Typescript implementation of differential dataflow. This allows the query results to update*incrementally*(rather than by re-running the whole query). This makes them blazing fast, usually sub-millisecond, even for highly complex queries.

Live queries support joins across collections. This allows you to:

1. load normalised data into collections and then de-normalise it through queries; simplifying your backend by avoiding the need for bespoke API endpoints that match your client
2. join data from multiple sources; for example, syncing some data out of a database, fetching some other data from an external API and then joining these into a unified data model for your front-end code

Every query returns another collection which can*also*be queried.

For more details on live queries, see the[Live Queries](/db/latest/docs/guides/live-queries)documentation.
[Making optimistic mutations](#making-optimistic-mutations)

Collections supportinsert,updateanddeleteoperations. When called, by default they trigger the correspondingonInsert,onUpdate,onDeletehandlers which are responsible for writing the mutation to the backend.
ts

```
// Define collection with persistence handlers
const todoCollection = createCollection({
  id: "todos",
  // ... other config
  onUpdate: async ({ transaction }) => {
    const { original, changes } = transaction.mutations[0]
    await api.todos.update(original.id, changes)
  },
})

// Immediately applies optimistic state
todoCollection.update(todo.id, (draft) => {
  draft.completed = true
})

```

```
// Define collection with persistence handlers
const todoCollection = createCollection({
  id: "todos",
  // ... other config
  onUpdate: async ({ transaction }) => {
    const { original, changes } = transaction.mutations[0]
    await api.todos.update(original.id, changes)
  },
})

// Immediately applies optimistic state
todoCollection.update(todo.id, (draft) => {
  draft.completed = true
})

```

Rather than mutating the collection data directly, the collection internally treats its synced/loaded data as immutable and maintains a separate set of local mutations as optimistic state. When live queries read from the collection, they see a local view that overlays the local optimistic mutations on-top-of the immutable synced data.

The optimistic state is held until theonUpdate(in this case) handler resolves - at which point the data is persisted to the server and synced back to the local collection.

If the handler throws an error, the optimistic state is rolled back.
[Explicit transactions](#explicit-transactions)

Mutations are based on aTransactionprimitive.

For simple state changes, directly mutating the collection and persisting with the operator handlers is enough.

But for more complex use cases, you can directly create custom actions withcreateOptimisticActionor custom transactions withcreateTransaction. This lets you do things such as do transactions with multiple mutations across multiple collections, do chained transactions w/ intermediate rollbacks, etc.

For example, in the following code, the mutationFn first sends the write to the server usingawait api.todos.update(updatedTodo)and then callsawait collection.refetch()to trigger a re-fetch of the collection contents using TanStack Query. When this second await resolves, the collection is up-to-date with the latest changes and the optimistic state is safely discarded.
ts

```
const updateTodo = createOptimisticAction<{ id: string }>({
  onMutate,
  mutationFn: async ({ transaction }) => {
    const { collection, modified: updatedTodo } = transaction.mutations[0]

    await api.todos.update(updatedTodo)
    await collection.refetch()
  },
})

```

```
const updateTodo = createOptimisticAction<{ id: string }>({
  onMutate,
  mutationFn: async ({ transaction }) => {
    const { collection, modified: updatedTodo } = transaction.mutations[0]

    await api.todos.update(updatedTodo)
    await collection.refetch()
  },
})

```[Uni-directional data flow](#uni-directional-data-flow)

This combines to support a model of uni-directional data flow, extending the redux/flux style state management pattern beyond the client, to take in the server as well:

With an instant inner loop of optimistic state, superseded in time by the slower outer loop of persisting to the server and syncing the updated server state back into the collection.
[API reference](#api-reference)[Collections](#collections)

There are a number of built-in collection types:

1. [QueryCollection](#querycollection)to load data into collections using[TanStack Query](https://tanstack.com/query)
2. [ElectricCollection](#electriccollection)to sync data into collections using[ElectricSQL](https://electric-sql.com)
3. [TrailBaseCollection](#trailbasecollection)to sync data into collections using[TrailBase](https://trailbase.io)
4. [LocalStorageCollection](#localstoragecollection)for small amounts of local-only state that syncs across browser tabs
5. [LocalOnlyCollection](#localonlycollection)for in-memory client data or UI state

You can also use:

- use live collection queries to[derive collections from other collections](#derived-collections)
- the[base Collection](#base-collection)to define your own collection types
[Collection schemas](#collection-schemas)

All collections optionally (though strongly recommended) support adding aschema.

If provided, this must be a[Standard Schema](https://standardschema.dev)compatible schema instance, such as a[Zod](https://zod.dev)or[Effect](https://effect.website/docs/schema/introduction/)schema.

The collection will use the schema to do client-side validation of optimistic mutations.

The collection will use the schema for its type so if you provide a schema, you can't also pass in an explicit
type (e.g.createCollection<Todo>()).
[QueryCollection](#querycollection)

[TanStack Query](https://tanstack.com/query)fetches data using managed queries. UsequeryCollectionOptionsto fetch data into a collection using TanStack Query:
ts

```
import { createCollection } from "@tanstack/react-db"
import { queryCollectionOptions } from "@tanstack/query-db-collection"

const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ["todoItems"],
    queryFn: async () => {
      const response = await fetch("/api/todos");
      return response.json();
    },
    getKey: (item) => item.id,
    schema: todoSchema, // any standard schema
  })
)

```

```
import { createCollection } from "@tanstack/react-db"
import { queryCollectionOptions } from "@tanstack/query-db-collection"

const todoCollection = createCollection(
  queryCollectionOptions({
    queryKey: ["todoItems"],
    queryFn: async () => {
      const response = await fetch("/api/todos");
      return response.json();
    },
    getKey: (item) => item.id,
    schema: todoSchema, // any standard schema
  })
)

```

The collection will be populated with the query results.
[ElectricCollection](#electriccollection)

[Electric](https://electric-sql.com)is a read-path sync engine for Postgres. It allows you to sync subsets of data out of a Postgres database,[through your API](https://electric-sql.com/blog/2024/11/21/local-first-with-your-existing-api), into a TanStack DB collection.

Electric's main primitive for sync is a[Shape](https://electric-sql.com/docs/guides/shapes). UseelectricCollectionOptionsto sync a shape into a collection:
ts

```
import { createCollection } from "@tanstack/react-db"
import { electricCollectionOptions } from "@tanstack/electric-db-collection"

export const todoCollection = createCollection(
  electricCollectionOptions({
    id: "todos",
    shapeOptions: {
      url: "https://example.com/v1/shape",
      params: {
        table: "todos",
      },
    },
    getKey: (item) => item.id,
    schema: todoSchema,
  })
)

```

```
import { createCollection } from "@tanstack/react-db"
import { electricCollectionOptions } from "@tanstack/electric-db-collection"

export const todoCollection = createCollection(
  electricCollectionOptions({
    id: "todos",
    shapeOptions: {
      url: "https://example.com/v1/shape",
      params: {
        table: "todos",
      },
    },
    getKey: (item) => item.id,
    schema: todoSchema,
  })
)

```

The Electric collection requires two Electric-specific options:

- shapeOptions— the Electric[ShapeStreamOptions](https://electric-sql.com/docs/api/clients/typescript#options)that define the[Shape](https://electric-sql.com/docs/guides/shapes)to sync into the collection; this includes the

- urlto your sync engine; and
- paramsto specify thetableto sync and any optionalwhereclauses, etc.
- urlto your sync engine; and
- paramsto specify thetableto sync and any optionalwhereclauses, etc.
- getKey— identifies the id for the rows being synced into the collection

A new collections doesn't start syncing until you callcollection.preload()or you query it.

Electric shapes allow you to filter data using where clauses:
ts

```
export const myPendingTodos = createCollection(
  electricCollectionOptions({
    id: "todos",
    shapeOptions: {
      url: "https://example.com/v1/shape",
      params: {
        table: "todos",
        where: `
        status = 'pending'
        AND
        user_id = '${user.id}'
      `,
      },
    },
    getKey: (item) => item.id,
    schema: todoSchema,
  })
)

```

```
export const myPendingTodos = createCollection(
  electricCollectionOptions({
    id: "todos",
    shapeOptions: {
      url: "https://example.com/v1/shape",
      params: {
        table: "todos",
        where: `
        status = 'pending'
        AND
        user_id = '${user.id}'
      `,
      },
    },
    getKey: (item) => item.id,
    schema: todoSchema,
  })
)

```Tip

Shape where clauses, used to filter the data you sync intoElectricCollections, are different from the[live queries](#live-queries)you use to query data in components.

Live queries are much more expressive than shapes, allowing you to query across collections, join, aggregate, etc. Shapes just contain filtered database tables and are used to populate the data in a collection.

If you need more control over what data syncs into the collection, Electric allows you to[use your API](https://electric-sql.com/blog/2024/11/21/local-first-with-your-existing-api#filtering)as a proxy to both authorize and filter data.

See the[Electric docs](https://electric-sql.com/docs/intro)for more information.
[TrailBaseCollection](#trailbasecollection)

[TrailBase](https://trailbase.io)is an easy-to-self-host, single-executable application backend with built-in SQLite, a V8 JS runtime, auth, admin UIs and sync functionality.

TrailBase lets you expose tables via[Record APIs](https://trailbase.io/documentation/apis_record/)and subscribe to changes whenenable_subscriptionsis set. UsetrailBaseCollectionOptionsto sync records into a collection:
ts

```
import { createCollection } from "@tanstack/react-db"
import { trailBaseCollectionOptions } from "@tanstack/trailbase-db-collection"
import { initClient } from "trailbase"

const trailBaseClient = initClient(`https://trailbase.io`)

export const todoCollection = createCollection<SelectTodo, Todo>(
  electricCollectionOptions({
    id: "todos",
    recordApi: trailBaseClient.records(`todos`),
    getKey: (item) => item.id,
    schema: todoSchema,
    parse: {
      created_at: (ts) => new Date(ts * 1000),
    },
    serialize: {
      created_at: (date) => Math.floor(date.valueOf() / 1000),
    },
  })
)

```

```
import { createCollection } from "@tanstack/react-db"
import { trailBaseCollectionOptions } from "@tanstack/trailbase-db-collection"
import { initClient } from "trailbase"

const trailBaseClient = initClient(`https://trailbase.io`)

export const todoCollection = createCollection<SelectTodo, Todo>(
  electricCollectionOptions({
    id: "todos",
    recordApi: trailBaseClient.records(`todos`),
    getKey: (item) => item.id,
    schema: todoSchema,
    parse: {
      created_at: (ts) => new Date(ts * 1000),
    },
    serialize: {
      created_at: (date) => Math.floor(date.valueOf() / 1000),
    },
  })
)

```

This collection requires the following TrailBase-specific options:

- recordApi— identifies the API to sync.
- getKey— identifies the id for the records being synced into the collection.
- parse— maps(v: Todo[k]) => SelectTodo[k].
- serialize— maps(v: SelectTodo[k]) => Todo[k].

A new collections doesn't start syncing until you callcollection.preload()or you query it.
[LocalStorageCollection](#localstoragecollection)

localStorage collections store small amounts of local-only state that persists across browser sessions and syncs across browser tabs in real-time. All data is stored under a single localStorage key and automatically synchronized using storage events.

UselocalStorageCollectionOptionsto create a collection that stores data in localStorage:
ts

```
import { createCollection } from "@tanstack/react-db"
import { localStorageCollectionOptions } from "@tanstack/react-db"

export const userPreferencesCollection = createCollection(
  localStorageCollectionOptions({
    id: "user-preferences",
    storageKey: "app-user-prefs", // localStorage key
    getKey: (item) => item.id,
    schema: userPrefsSchema,
  })
)

```

```
import { createCollection } from "@tanstack/react-db"
import { localStorageCollectionOptions } from "@tanstack/react-db"

export const userPreferencesCollection = createCollection(
  localStorageCollectionOptions({
    id: "user-preferences",
    storageKey: "app-user-prefs", // localStorage key
    getKey: (item) => item.id,
    schema: userPrefsSchema,
  })
)

```

The localStorage collection requires:

- storageKey— the localStorage key where all collection data is stored
- getKey— identifies the id for items in the collection

Mutation handlers (onInsert,onUpdate,onDelete) are completely optional. Data will persist to localStorage whether or not you provide handlers. You can provide alternative storage backends likesessionStorageor custom implementations that match the localStorage API.
ts

```
export const sessionCollection = createCollection(
  localStorageCollectionOptions({
    id: "session-data",
    storageKey: "session-key",
    storage: sessionStorage, // Use sessionStorage instead
    getKey: (item) => item.id,
  })
)

```

```
export const sessionCollection = createCollection(
  localStorageCollectionOptions({
    id: "session-data",
    storageKey: "session-key",
    storage: sessionStorage, // Use sessionStorage instead
    getKey: (item) => item.id,
  })
)

```Tip

localStorage collections are perfect for user preferences, UI state, and other data that should persist locally but doesn't need server synchronization. For server-synchronized data, use[QueryCollection](#querycollection)or[ElectricCollection](#electriccollection)instead.[LocalOnlyCollection](#localonlycollection)

LocalOnly collections are designed for in-memory client data or UI state that doesn't need to persist across browser sessions or sync across tabs. They provide a simple way to manage temporary, session-only data with full optimistic mutation support.

UselocalOnlyCollectionOptionsto create a collection that stores data only in memory:
ts

```
import { createCollection } from "@tanstack/react-db"
import { localOnlyCollectionOptions } from "@tanstack/react-db"

export const uiStateCollection = createCollection(
  localOnlyCollectionOptions({
    id: "ui-state",
    getKey: (item) => item.id,
    schema: uiStateSchema,
    // Optional initial data to populate the collection
    initialData: [
      { id: "sidebar", isOpen: false },
      { id: "theme", mode: "light" },
    ],
  })
)

```

```
import { createCollection } from "@tanstack/react-db"
import { localOnlyCollectionOptions } from "@tanstack/react-db"

export const uiStateCollection = createCollection(
  localOnlyCollectionOptions({
    id: "ui-state",
    getKey: (item) => item.id,
    schema: uiStateSchema,
    // Optional initial data to populate the collection
    initialData: [
      { id: "sidebar", isOpen: false },
      { id: "theme", mode: "light" },
    ],
  })
)

```

The LocalOnly collection requires:

- getKey— identifies the id for items in the collection

Optional configuration:

- initialData— array of items to populate the collection with on creation
- onInsert,onUpdate,onDelete— optional mutation handlers for custom logic

Mutation handlers are completely optional. When provided, they are called before the optimistic state is confirmed. The collection automatically manages the transition from optimistic to confirmed state internally.
ts

```
export const tempDataCollection = createCollection(
  localOnlyCollectionOptions({
    id: "temp-data",
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      // Custom logic before confirming the insert
      console.log("Inserting:", transaction.mutations[0].modified)
    },
    onUpdate: async ({ transaction }) => {
      // Custom logic before confirming the update
      const { original, modified } = transaction.mutations[0]
      console.log("Updating from", original, "to", modified)
    },
  })
)

```

```
export const tempDataCollection = createCollection(
  localOnlyCollectionOptions({
    id: "temp-data",
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      // Custom logic before confirming the insert
      console.log("Inserting:", transaction.mutations[0].modified)
    },
    onUpdate: async ({ transaction }) => {
      // Custom logic before confirming the update
      const { original, modified } = transaction.mutations[0]
      console.log("Updating from", original, "to", modified)
    },
  })
)

```Tip

LocalOnly collections are perfect for temporary UI state, form data, or any client-side data that doesn't need persistence. For data that should persist across sessions, use[LocalStorageCollection](#localstoragecollection)instead.[Derived collections](#derived-collections)

Live queries return collections. This allows you to derive collections from other collections.

For example:
ts

```
import { createLiveQueryCollection, eq } from "@tanstack/db"

// Imagine you have a collection of todos.
const todoCollection = createCollection({
  // config
})

// You can derive a new collection that's a subset of it.
const completedTodoCollection = createLiveQueryCollection({
  startSync: true,
  query: (q) =>
    q.from({ todo: todoCollection }).where(({ todo }) => todo.completed),
})

```

```
import { createLiveQueryCollection, eq } from "@tanstack/db"

// Imagine you have a collection of todos.
const todoCollection = createCollection({
  // config
})

// You can derive a new collection that's a subset of it.
const completedTodoCollection = createLiveQueryCollection({
  startSync: true,
  query: (q) =>
    q.from({ todo: todoCollection }).where(({ todo }) => todo.completed),
})

```

This also works with joins to derive collections from multiple source collections. And it works recursively -- you can derive collections from other derived collections. Changes propagate efficiently using differential dataflow and it's collections all the way down.
[Collection](#collection)

There is aCollectioninterface in[../packages/db/src/collection.ts](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts). You can use this to implement your own collection types.

See the existing implementations in[../packages/db](https://github.com/TanStack/db/tree/main/packages/db),[../packages/query-db-collection](https://github.com/TanStack/db/tree/main/packages/query-db-collection),[../packages/electric-db-collection](https://github.com/TanStack/db/tree/main/packages/electric-db-collection)and[../packages/trailbase-db-collection](https://github.com/TanStack/db/tree/main/packages/trailbase-db-collection)for reference.
[Live queries](#live-queries)[useLiveQuery hook](#uselivequery-hook)

Use theuseLiveQueryhook to assign live query results to a state variable in your React components:
ts

```
import { useLiveQuery } from '@tanstack/react-db'
import { eq } from '@tanstack/db'

const Todos = () => {
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todo: todoCollection })
      .where(({ todo }) => eq(todo.completed, false))
      .orderBy(({ todo }) => todo.created_at, 'asc')
      .select(({ todo }) => ({
        id: todo.id,
        text: todo.text
      }))
  )

  return <List items={ todos } />
}

```

```
import { useLiveQuery } from '@tanstack/react-db'
import { eq } from '@tanstack/db'

const Todos = () => {
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todo: todoCollection })
      .where(({ todo }) => eq(todo.completed, false))
      .orderBy(({ todo }) => todo.created_at, 'asc')
      .select(({ todo }) => ({
        id: todo.id,
        text: todo.text
      }))
  )

  return <List items={ todos } />
}

```

You can also query across collections with joins:
ts

```
import { useLiveQuery } from '@tanstack/react-db'
import { eq } from '@tanstack/db'

const Todos = () => {
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todos: todoCollection })
      .join(
        { lists: listCollection },
        ({ todos, lists }) => eq(lists.id, todos.listId),
        'inner'
      )
      .where(({ lists }) => eq(lists.active, true))
      .select(({ todos, lists }) => ({
        id: todos.id,
        title: todos.title,
        listName: lists.name
      }))
  )

  return <List items={ todos } />
}

```

```
import { useLiveQuery } from '@tanstack/react-db'
import { eq } from '@tanstack/db'

const Todos = () => {
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todos: todoCollection })
      .join(
        { lists: listCollection },
        ({ todos, lists }) => eq(lists.id, todos.listId),
        'inner'
      )
      .where(({ lists }) => eq(lists.active, true))
      .select(({ todos, lists }) => ({
        id: todos.id,
        title: todos.title,
        listName: lists.name
      }))
  )

  return <List items={ todos } />
}

```[queryBuilder](#querybuilder)

You can also build queries directly (outside of the component lifecycle) using the underlyingqueryBuilderAPI:
ts

```
import { createLiveQueryCollection, eq } from "@tanstack/db"

const completedTodos = createLiveQueryCollection({
  startSync: true,
  query: (q) =>
    q
      .from({ todo: todoCollection })
      .where(({ todo }) => eq(todo.completed, true)),
})

const results = completedTodos.toArray

```

```
import { createLiveQueryCollection, eq } from "@tanstack/db"

const completedTodos = createLiveQueryCollection({
  startSync: true,
  query: (q) =>
    q
      .from({ todo: todoCollection })
      .where(({ todo }) => eq(todo.completed, true)),
})

const results = completedTodos.toArray

```

Note also that:

1. the query results[are themselves a collection](#derived-collections)
2. theuseLiveQueryautomatically starts and stops live query subscriptions when you mount and unmount your components; if you're creating queries manually, you need to manually manage the subscription lifecycle yourself

See the[Live Queries](/db/latest/docs/guides/live-queries)documentation for more details.
[Transactional mutators](#transactional-mutators)

Transactional mutators allow you to batch and stage local changes across collections with:

- immediate application of local optimistic updates
- flexible mutationFns to handle writes, with automatic rollbacks and management of optimistic state
[mutationFn](#mutationfn)

Mutators are created with amutationFn. You can define a single, genericmutationFnfor your whole app. Or you can define collection or mutation specific functions.

ThemutationFnis responsible for handling the local changes and processing them, usually to send them to a server or database to be stored, e.g.:
tsx

```
import type { MutationFn } from "@tanstack/react-db"

const mutationFn: MutationFn = async ({ transaction }) => {
  const response = await api.todos.create(transaction.mutations)

  if (!response.ok) {
    // Throwing an error will rollback the optimistic state.
    throw new Error(`HTTP Error: ${response.status}`)
  }

  const result = await response.json()

  // Wait for the transaction to be synced back from the server
  // before discarding the optimistic state.
  const collection: Collection = transaction.mutations[0].collection
  await collection.refetch()
}

```

```
import type { MutationFn } from "@tanstack/react-db"

const mutationFn: MutationFn = async ({ transaction }) => {
  const response = await api.todos.create(transaction.mutations)

  if (!response.ok) {
    // Throwing an error will rollback the optimistic state.
    throw new Error(`HTTP Error: ${response.status}`)
  }

  const result = await response.json()

  // Wait for the transaction to be synced back from the server
  // before discarding the optimistic state.
  const collection: Collection = transaction.mutations[0].collection
  await collection.refetch()
}

```[createOptimisticAction](#createoptimisticaction)

UsecreateOptimisticActionwith yourmutationFnandonMutatefunctions to create an action that you can use to mutate data in your components in fully custom ways:
tsx

```
import { createOptimisticAction } from "@tanstack/react-db"

// Create the `addTodo` action, passing in your `mutationFn` and `onMutate`.
const addTodo = createOptimisticAction<string>({
  onMutate: (text) => {
    // Instantly applies the local optimistic state.
    todoCollection.insert({
      id: uuid(),
      text,
      completed: false,
    })
  },
  mutationFn: async (text) => {
    // Persist the todo to your backend
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify({ text, completed: false }),
    })
    return response.json()
  },
})

const Todo = () => {
  const handleClick = () => {
    // Triggers the onMutate and then the mutationFn
    addTodo("🔥 Make app faster")
  }

  return <Button onClick={handleClick} />
}

```

```
import { createOptimisticAction } from "@tanstack/react-db"

// Create the `addTodo` action, passing in your `mutationFn` and `onMutate`.
const addTodo = createOptimisticAction<string>({
  onMutate: (text) => {
    // Instantly applies the local optimistic state.
    todoCollection.insert({
      id: uuid(),
      text,
      completed: false,
    })
  },
  mutationFn: async (text) => {
    // Persist the todo to your backend
    const response = await fetch("/api/todos", {
      method: "POST",
      body: JSON.stringify({ text, completed: false }),
    })
    return response.json()
  },
})

const Todo = () => {
  const handleClick = () => {
    // Triggers the onMutate and then the mutationFn
    addTodo("🔥 Make app faster")
  }

  return <Button onClick={handleClick} />
}

```[Manual Transactions](#manual-transactions)

By manually creating transactions, you can fully control their lifecycles and behaviors.createOptimisticActionis a ~25 line
function which implements a common transaction pattern. Feel free to invent your own patterns!

Here's one way you could use transactions.
ts

```
import { createTransaction } from "@tanstack/react-db"

const addTodoTx = createTransaction({
  autoCommit: false,
  mutationFn: async ({ transaction }) => {
    // Persist data to backend
    await Promise.all(transaction.mutations.map(mutation => {
      return await api.saveTodo(mutation.modified)
    })
  },
})

// Apply first change
addTodoTx.mutate(() => todoCollection.insert({ id: '1', text: 'First todo', completed: false }))

// user reviews change

// Apply another change
addTodoTx.mutate(() => todoCollection.insert({ id: '2', text: 'Second todo', completed: false }))

// User decides to save and we call .commit() and the mutations are persisted to the backend.
addTodoTx.commit()

```

```
import { createTransaction } from "@tanstack/react-db"

const addTodoTx = createTransaction({
  autoCommit: false,
  mutationFn: async ({ transaction }) => {
    // Persist data to backend
    await Promise.all(transaction.mutations.map(mutation => {
      return await api.saveTodo(mutation.modified)
    })
  },
})

// Apply first change
addTodoTx.mutate(() => todoCollection.insert({ id: '1', text: 'First todo', completed: false }))

// user reviews change

// Apply another change
addTodoTx.mutate(() => todoCollection.insert({ id: '2', text: 'Second todo', completed: false }))

// User decides to save and we call .commit() and the mutations are persisted to the backend.
addTodoTx.commit()

```[Transaction lifecycle](#transaction-lifecycle)

Transactions progress through the following states:

1. pending: Initial state when a transaction is created and optimistic mutations can be applied
2. persisting: Transaction is being persisted to the backend
3. completed: Transaction has been successfully persisted and any backend changes have been synced back.
4. failed: An error was thrown while persisting or syncing back the Transaction
[Write operations](#write-operations)

Collections supportinsert,updateanddeleteoperations.
[insert](#insert)typescript

```
// Insert a single item
myCollection.insert({ text: "Buy groceries", completed: false })

// Insert multiple items
insert([
  { text: "Buy groceries", completed: false },
  { text: "Walk dog", completed: false },
])

// Insert with optimistic updates disabled
myCollection.insert(
  { text: "Server-validated item", completed: false },
  { optimistic: false }
)

// Insert with metadata and optimistic control
myCollection.insert(
  { text: "Custom item", completed: false },
  {
    metadata: { source: "import" },
    optimistic: true, // default behavior
  }
)

```

```
// Insert a single item
myCollection.insert({ text: "Buy groceries", completed: false })

// Insert multiple items
insert([
  { text: "Buy groceries", completed: false },
  { text: "Walk dog", completed: false },
])

// Insert with optimistic updates disabled
myCollection.insert(
  { text: "Server-validated item", completed: false },
  { optimistic: false }
)

// Insert with metadata and optimistic control
myCollection.insert(
  { text: "Custom item", completed: false },
  {
    metadata: { source: "import" },
    optimistic: true, // default behavior
  }
)

```[update](#update)

We use a proxy to capture updates as immutable draft optimistic updates.
typescript

```
// Update a single item
update(todo.id, (draft) => {
  draft.completed = true
})

// Update multiple items
update([todo1.id, todo2.id], (drafts) => {
  drafts.forEach((draft) => {
    draft.completed = true
  })
})

// Update with metadata
update(todo.id, { metadata: { reason: "user update" } }, (draft) => {
  draft.text = "Updated text"
})

// Update without optimistic updates
update(todo.id, { optimistic: false }, (draft) => {
  draft.status = "server-validated"
})

// Update with both metadata and optimistic control
update(
  todo.id,
  {
    metadata: { reason: "admin update" },
    optimistic: false,
  },
  (draft) => {
    draft.priority = "high"
  }
)

```

```
// Update a single item
update(todo.id, (draft) => {
  draft.completed = true
})

// Update multiple items
update([todo1.id, todo2.id], (drafts) => {
  drafts.forEach((draft) => {
    draft.completed = true
  })
})

// Update with metadata
update(todo.id, { metadata: { reason: "user update" } }, (draft) => {
  draft.text = "Updated text"
})

// Update without optimistic updates
update(todo.id, { optimistic: false }, (draft) => {
  draft.status = "server-validated"
})

// Update with both metadata and optimistic control
update(
  todo.id,
  {
    metadata: { reason: "admin update" },
    optimistic: false,
  },
  (draft) => {
    draft.priority = "high"
  }
)

```[delete](#delete)typescript

```
// Delete a single item
delete todo.id

// Delete multiple items
delete [todo1.id, todo2.id]

// Delete with metadata
delete (todo.id, { metadata: { reason: "completed" } })

// Delete without optimistic updates (waits for server confirmation)
delete (todo.id, { optimistic: false })

// Delete with metadata and optimistic control
delete (todo.id,
{
  metadata: { reason: "admin deletion" },
  optimistic: false,
})

```

```
// Delete a single item
delete todo.id

// Delete multiple items
delete [todo1.id, todo2.id]

// Delete with metadata
delete (todo.id, { metadata: { reason: "completed" } })

// Delete without optimistic updates (waits for server confirmation)
delete (todo.id, { optimistic: false })

// Delete with metadata and optimistic control
delete (todo.id,
{
  metadata: { reason: "admin deletion" },
  optimistic: false,
})

```[Controlling optimistic behavior](#controlling-optimistic-behavior)

By default, all mutations (insert,update,delete) apply optimistic updates immediately to provide instant feedback in your UI. However, there are cases where you may want to disable this behavior and wait for server confirmation before applying changes locally.
[When to use optimistic: false](#when-to-use-optimistic-false)

Consider disabling optimistic updates when:

- **Complex server-side processing**: Inserts that depend on server-side generation (e.g., cascading foreign keys, computed fields)
- **Validation requirements**: Operations where backend validation might reject the change
- **Confirmation workflows**: Deletes where UX should wait for confirmation before removing data
- **Batch operations**: Large operations where optimistic rollback would be disruptive
[Behavior differences](#behavior-differences)

**optimistic: true (default)**:

- Immediately applies mutation to the local store
- Provides instant UI feedback
- Requires rollback if server rejects the mutation
- Best for simple, predictable operations

**optimistic: false**:

- Does not modify local store until server confirms
- No immediate UI feedback, but no rollback needed
- UI updates only after successful server response
- Best for complex or validation-heavy operations
typescript

```
// Example: Critical deletion that needs confirmation
const handleDeleteAccount = () => {
  // Don't remove from UI until server confirms
  userCollection.delete(userId, { optimistic: false })
}

// Example: Server-generated data
const handleCreateInvoice = () => {
  // Server generates invoice number, tax calculations, etc.
  invoiceCollection.insert(invoiceData, { optimistic: false })
}

// Example: Mixed approach in same transaction
tx.mutate(() => {
  // Instant UI feedback for simple change
  todoCollection.update(todoId, (draft) => {
    draft.completed = true
  })

  // Wait for server confirmation for complex change
  auditCollection.insert(auditRecord, { optimistic: false })
})

```

```
// Example: Critical deletion that needs confirmation
const handleDeleteAccount = () => {
  // Don't remove from UI until server confirms
  userCollection.delete(userId, { optimistic: false })
}

// Example: Server-generated data
const handleCreateInvoice = () => {
  // Server generates invoice number, tax calculations, etc.
  invoiceCollection.insert(invoiceData, { optimistic: false })
}

// Example: Mixed approach in same transaction
tx.mutate(() => {
  // Instant UI feedback for simple change
  todoCollection.update(todoId, (draft) => {
    draft.completed = true
  })

  // Wait for server confirmation for complex change
  auditCollection.insert(auditRecord, { optimistic: false })
})

```[Usage examples](#usage-examples)

Here we illustrate two common ways of using TanStack DB:

1. [using TanStack Query](#1-tanstack-query)with an existing REST API
2. [using the ElectricSQL sync engine](#2-electricsql-sync)with a generic ingestion endpoint
Tip

You can combine these patterns. One of the benefits of TanStack DB is that you can integrate different ways of loading data and handling mutations into the same app. Your components don't need to know where the data came from or goes.[1. TanStack Query](#1-tanstack-query)

You can use TanStack DB with your existing REST API via TanStack Query.

The steps are to:

1. create[QueryCollection](#querycollection)s that load data using TanStack Query
2. implement[mutationFn](#mutationfn)s that handle mutations by posting them to your API endpoints
tsx

```
import { useLiveQuery, createCollection } from "@tanstack/react-db"
import { queryCollectionOptions } from "@tanstack/query-db-collection"

// Load data into collections using TanStack Query.
// It's common to define these in a `collections` module.
const todoCollection = createCollection(queryCollectionOptions({
  queryKey: ["todos"],
  queryFn: async () => fetch("/api/todos"),
  getKey: (item) => item.id,
  schema: todoSchema, // any standard schema
  onInsert: async ({ transaction }) => {
    const { changes: newTodo } = transaction.mutations[0]

    // Handle the local write by sending it to your API.
    await api.todos.create(newTodo)
  }
  // also add onUpdate, onDelete as needed.
}))
const listCollection = createCollection(queryCollectionOptions({
  queryKey: ["todo-lists"],
  queryFn: async () => fetch("/api/todo-lists"),
  getKey: (item) => item.id,
  schema: todoListSchema,
  onInsert: async ({ transaction }) => {
    const { changes: newTodo } = transaction.mutations[0]

    // Handle the local write by sending it to your API.
    await api.todoLists.create(newTodo)
  }
  // also add onUpdate, onDelete as needed.
}))

const Todos = () => {
  // Read the data using live queries. Here we show a live
  // query that joins across two collections.
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todo: todoCollection })
      .join(
        { list: listCollection },
        ({ todo, list }) => eq(list.id, todo.list_id),
        'inner'
      )
      .where(({ list }) => eq(list.active, true))
      .select(({ todo, list }) => ({
        id: todo.id,
        text: todo.text,
        status: todo.status,
        listName: list.name
      }))
  )

  // ...

}

```

```
import { useLiveQuery, createCollection } from "@tanstack/react-db"
import { queryCollectionOptions } from "@tanstack/query-db-collection"

// Load data into collections using TanStack Query.
// It's common to define these in a `collections` module.
const todoCollection = createCollection(queryCollectionOptions({
  queryKey: ["todos"],
  queryFn: async () => fetch("/api/todos"),
  getKey: (item) => item.id,
  schema: todoSchema, // any standard schema
  onInsert: async ({ transaction }) => {
    const { changes: newTodo } = transaction.mutations[0]

    // Handle the local write by sending it to your API.
    await api.todos.create(newTodo)
  }
  // also add onUpdate, onDelete as needed.
}))
const listCollection = createCollection(queryCollectionOptions({
  queryKey: ["todo-lists"],
  queryFn: async () => fetch("/api/todo-lists"),
  getKey: (item) => item.id,
  schema: todoListSchema,
  onInsert: async ({ transaction }) => {
    const { changes: newTodo } = transaction.mutations[0]

    // Handle the local write by sending it to your API.
    await api.todoLists.create(newTodo)
  }
  // also add onUpdate, onDelete as needed.
}))

const Todos = () => {
  // Read the data using live queries. Here we show a live
  // query that joins across two collections.
  const { data: todos } = useLiveQuery((q) =>
    q
      .from({ todo: todoCollection })
      .join(
        { list: listCollection },
        ({ todo, list }) => eq(list.id, todo.list_id),
        'inner'
      )
      .where(({ list }) => eq(list.active, true))
      .select(({ todo, list }) => ({
        id: todo.id,
        text: todo.text,
        status: todo.status,
        listName: list.name
      }))
  )

  // ...

}

```

This pattern allows you to extend an existing TanStack Query application, or any application built on a REST API, with blazing fast, cross-collection live queries and local optimistic mutations with automatically managed optimistic state.
[2. ElectricSQL sync](#2-electricsql-sync)

One of the most powerful ways of using TanStack DB is with a sync engine, for a fully local-first experience with real-time sync. This allows you to incrementally adopt sync into an existing app, whilst still handling writes with your existing API.

Here, we illustrate this pattern using[ElectricSQL](https://electric-sql.com)as the sync engine.
tsx

```
import type { Collection } from '@tanstack/db'
import type { MutationFn, PendingMutation, createCollection } from '@tanstack/react-db'
import { electricCollectionOptions } from '@tanstack/electric-db-collection'

export const todoCollection = createCollection(electricCollectionOptions({
  id: 'todos',
  schema: todoSchema,
  // Electric syncs data using "shapes". These are filtered views
  // on database tables that Electric keeps in sync for you.
  shapeOptions: {
    url: 'https://api.electric-sql.cloud/v1/shape',
    params: {
      table: 'todos'
    }
  },
  getKey: (item) => item.id,
  schema: todoSchema,
  onInsert: async ({ transaction }) => {
    const response = await api.todos.create(transaction.mutations[0].modified)

    return { txid: response.txid}
  }
  // You can also implement onUpdate, onDelete as needed.
}))

const AddTodo = () => {
  return (
    <Button
      onClick={() =>
        todoCollection.insert({ text: "🔥 Make app faster" })
      }
    />
  )
}

```

```
import type { Collection } from '@tanstack/db'
import type { MutationFn, PendingMutation, createCollection } from '@tanstack/react-db'
import { electricCollectionOptions } from '@tanstack/electric-db-collection'

export const todoCollection = createCollection(electricCollectionOptions({
  id: 'todos',
  schema: todoSchema,
  // Electric syncs data using "shapes". These are filtered views
  // on database tables that Electric keeps in sync for you.
  shapeOptions: {
    url: 'https://api.electric-sql.cloud/v1/shape',
    params: {
      table: 'todos'
    }
  },
  getKey: (item) => item.id,
  schema: todoSchema,
  onInsert: async ({ transaction }) => {
    const response = await api.todos.create(transaction.mutations[0].modified)

    return { txid: response.txid}
  }
  // You can also implement onUpdate, onDelete as needed.
}))

const AddTodo = () => {
  return (
    <Button
      onClick={() =>
        todoCollection.insert({ text: "🔥 Make app faster" })
      }
    />
  )
}

```[React Native](#react-native)

When using TanStack DB with React Native, you need to install and configure a UUID generation library since React Native doesn't include crypto.randomUUID() by default.

Install thereact-native-random-uuidpackage:
bash

```
npm install react-native-random-uuid

```

```
npm install react-native-random-uuid

```

Then import it at the entry point of your React Native app (e.g., in yourApp.jsorindex.js):
javascript

```
import 'react-native-random-uuid'

```

```
import 'react-native-random-uuid'

```

This polyfill provides thecrypto.randomUUID()function that TanStack DB uses internally for generating unique identifiers.
[More info](#more-info)

If you have questions / need help using TanStack DB, let us know on the Discord or start a GitHub discussion:

- [#db channel in the TanStack discord](https://discord.gg/yjUNbvbraC)
- [GitHub discussions](https://github.com/tanstack/db/discussions)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/overview.md)[Discord](https://tlinz.com/discord)[Quick Start](/db/latest/docs/quick-start)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>