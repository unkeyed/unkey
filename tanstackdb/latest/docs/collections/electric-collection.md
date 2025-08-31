# Electric Collection | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageElectric CollectionOverviewInstallationBasic UsageConfiguration OptionsRequired OptionsOptionalPersistence HandlersPersistence HandlersElectric Proxy ExampleOptimistic Updates with Explicit TransactionsUtility Methods# Electric Collection

Copy Markdown[Electric Collection](#electric-collection)

Electric collections provide seamless integration between TanStack DB and ElectricSQL, enabling real-time data synchronization with your Postgres database through Electric's sync engine.
[Overview](#overview)

The@tanstack/electric-db-collectionpackage allows you to create collections that:

- Automatically sync data from Postgres via Electric shapes
- Support optimistic updates with transaction matching and automatic rollback on errors
- Handle persistence through customizable mutation handlers
[Installation](#installation)bash

```
npm install @tanstack/electric-db-collection @tanstack/react-db

```

```
npm install @tanstack/electric-db-collection @tanstack/react-db

```[Basic Usage](#basic-usage)typescript

```
import { createCollection } from '@tanstack/react-db'
import { electricCollectionOptions } from '@tanstack/electric-db-collection'

const todosCollection = createCollection(
  electricCollectionOptions({
    shapeOptions: {
      url: '/api/todos',
    },
    getKey: (item) => item.id,
  })
)

```

```
import { createCollection } from '@tanstack/react-db'
import { electricCollectionOptions } from '@tanstack/electric-db-collection'

const todosCollection = createCollection(
  electricCollectionOptions({
    shapeOptions: {
      url: '/api/todos',
    },
    getKey: (item) => item.id,
  })
)

```[Configuration Options](#configuration-options)

TheelectricCollectionOptionsfunction accepts the following options:
[Required Options](#required-options)

- shapeOptions: Configuration for the ElectricSQL ShapeStream

- url: The URL of your proxy to Electric
- url: The URL of your proxy to Electric
- getKey: Function to extract the unique key from an item
[Optional](#optional)

- id: Unique identifier for the collection
- schema: Schema for validating items. Any Standard Schema compatible schema
- sync: Custom sync configuration
[Persistence Handlers](#persistence-handlers)

- onInsert: Handler called before insert operations
- onUpdate: Handler called before update operations
- onDelete: Handler called before delete operations
[Persistence Handlers](#persistence-handlers-1)

Handlers can be defined to run on mutations. They are useful to send mutations to the backend and confirming them once Electric delivers the corresponding transactions. Until confirmation, TanStack DB blocks sync data for the collection to prevent race conditions. To avoid any delays, it’s important to use a matching strategy.

The most reliable strategy is for the backend to include the transaction ID (txid) in its response, allowing the client to match each mutation with Electric’s transaction identifiers for precise confirmation. If no strategy is provided, client mutations are automatically confirmed after three seconds.
typescript

```
const todosCollection = createCollection(
  electricCollectionOptions({
    id: 'todos',
    schema: todoSchema,
    getKey: (item) => item.id,
    shapeOptions: {
      url: '/api/todos',
      params: { table: 'todos' },
    },
    
    onInsert: async ({ transaction }) => {
      const newItem = transaction.mutations[0].modified
      const response = await api.todos.create(newItem)
      
      return { txid: response.txid }
    },
    
    // you can also implement onUpdate and onDelete handlers
  })
)

```

```
const todosCollection = createCollection(
  electricCollectionOptions({
    id: 'todos',
    schema: todoSchema,
    getKey: (item) => item.id,
    shapeOptions: {
      url: '/api/todos',
      params: { table: 'todos' },
    },
    
    onInsert: async ({ transaction }) => {
      const newItem = transaction.mutations[0].modified
      const response = await api.todos.create(newItem)
      
      return { txid: response.txid }
    },
    
    // you can also implement onUpdate and onDelete handlers
  })
)

```

On the backend, you can extract thetxidfor a transaction by querying Postgres directly.
ts

```
async function generateTxId(tx) {
  // The ::xid cast strips off the epoch, giving you the raw 32-bit value
  // that matches what PostgreSQL sends in logical replication streams
  // (and then exposed through Electric which we'll match against
  // in the client).
  const result = await tx.execute(
    sql`SELECT pg_current_xact_id()::xid::text as txid`
  )
  const txid = result.rows[0]?.txid

  if (txid === undefined) {
    throw new Error(`Failed to get transaction ID`)
  }

  return parseInt(txid as string, 10)
}

```

```
async function generateTxId(tx) {
  // The ::xid cast strips off the epoch, giving you the raw 32-bit value
  // that matches what PostgreSQL sends in logical replication streams
  // (and then exposed through Electric which we'll match against
  // in the client).
  const result = await tx.execute(
    sql`SELECT pg_current_xact_id()::xid::text as txid`
  )
  const txid = result.rows[0]?.txid

  if (txid === undefined) {
    throw new Error(`Failed to get transaction ID`)
  }

  return parseInt(txid as string, 10)
}

```[Electric Proxy Example](#electric-proxy-example)

Electric is typically deployed behind a proxy server that handles shape configuration, authentication and authorization. This provides better security and allows you to control what data users can access without exposing Electric to the client.

Here is an example proxy implementation using TanStack Starter:
js

```
import { createServerFileRoute } from "@tanstack/react-start/server"
import { ELECTRIC_PROTOCOL_QUERY_PARAMS } from "@electric-sql/client"

// Electric URL
const baseUrl = 'http://.../v1/shape'

const serve = async ({ request }: { request: Request }) => {
  // ...check user authorization  
  const url = new URL(request.url)
  const originUrl = new URL(baseUrl)

  // passthrough parameters from electric client
  url.searchParams.forEach((value, key) => {
    if (ELECTRIC_PROTOCOL_QUERY_PARAMS.includes(key)) {
      originUrl.searchParams.set(key, value)
    }
  })

  // set shape parameters 
  // full spec: https://github.com/electric-sql/electric/blob/main/website/electric-api.yaml
  originUrl.searchParams.set("table", "todos")
  // Where clause to filter rows in the table (optional).
  // originUrl.searchParams.set("where", "completed = true")
  
  // Select the columns to sync (optional)
  // originUrl.searchParams.set("columns", "id,text,completed")

  const response = await fetch(originUrl)
  const headers = new Headers(response.headers)
  headers.delete("content-encoding")
  headers.delete("content-length")

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}

export const ServerRoute = createServerFileRoute("/api/todos").methods({
  GET: serve,
})

```

```
import { createServerFileRoute } from "@tanstack/react-start/server"
import { ELECTRIC_PROTOCOL_QUERY_PARAMS } from "@electric-sql/client"

// Electric URL
const baseUrl = 'http://.../v1/shape'

const serve = async ({ request }: { request: Request }) => {
  // ...check user authorization  
  const url = new URL(request.url)
  const originUrl = new URL(baseUrl)

  // passthrough parameters from electric client
  url.searchParams.forEach((value, key) => {
    if (ELECTRIC_PROTOCOL_QUERY_PARAMS.includes(key)) {
      originUrl.searchParams.set(key, value)
    }
  })

  // set shape parameters 
  // full spec: https://github.com/electric-sql/electric/blob/main/website/electric-api.yaml
  originUrl.searchParams.set("table", "todos")
  // Where clause to filter rows in the table (optional).
  // originUrl.searchParams.set("where", "completed = true")
  
  // Select the columns to sync (optional)
  // originUrl.searchParams.set("columns", "id,text,completed")

  const response = await fetch(originUrl)
  const headers = new Headers(response.headers)
  headers.delete("content-encoding")
  headers.delete("content-length")

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}

export const ServerRoute = createServerFileRoute("/api/todos").methods({
  GET: serve,
})

```[Optimistic Updates with Explicit Transactions](#optimistic-updates-with-explicit-transactions)

For more advanced use cases, you can create custom actions that can do multiple mutations across collections transactionally. In this case, you need to explicitly await for the transaction ID usingutils.awaitTxId().
typescript

```
const addTodoAction = createOptimisticAction({
  onMutate: ({ text }) => {
    // optimistically insert with a temporary ID
    const tempId = crypto.randomUUID()
    todosCollection.insert({
      id: tempId,
      text,
      completed: false,
      created_at: new Date(),
    })
    
    // ... mutate other collections
  },
  
  mutationFn: async ({ text }) => {
    const response = await api.todos.create({
      data: { text, completed: false }
    })
    
    await todosCollection.utils.awaitTxId(response.txid)
  }
})

```

```
const addTodoAction = createOptimisticAction({
  onMutate: ({ text }) => {
    // optimistically insert with a temporary ID
    const tempId = crypto.randomUUID()
    todosCollection.insert({
      id: tempId,
      text,
      completed: false,
      created_at: new Date(),
    })
    
    // ... mutate other collections
  },
  
  mutationFn: async ({ text }) => {
    const response = await api.todos.create({
      data: { text, completed: false }
    })
    
    await todosCollection.utils.awaitTxId(response.txid)
  }
})

```[Utility Methods](#utility-methods)

The collection provides these utility methods viacollection.utils:

- awaitTxId(txid, timeout?): Manually wait for a specific transaction ID to be synchronized
typescript

```
todosCollection.utils.awaitTxId(12345)

```

```
todosCollection.utils.awaitTxId(12345)

```

This is useful when you need to ensure a mutation has been synchronized before proceeding with other operations.[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/collections/electric-collection.md)[Query Collection](/db/latest/docs/collections/query-collection)[Core API Reference](/db/latest/docs/reference/index)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>