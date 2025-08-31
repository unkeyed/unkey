# createCollection | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: createCollection()Type ParametersParametersoptionsReturnsExamples# createCollection

Copy Markdown[Function: createCollection()](#function-createcollection)ts

```
function createCollection<TExplicit, TKey, TUtils, TSchema, TFallback>(options): Collection<ResolveType<TExplicit, TSchema, TFallback>, TKey, TUtils, TSchema, ResolveInsertInput<TExplicit, TSchema, TFallback>>

```

```
function createCollection<TExplicit, TKey, TUtils, TSchema, TFallback>(options): Collection<ResolveType<TExplicit, TSchema, TFallback>, TKey, TUtils, TSchema, ResolveInsertInput<TExplicit, TSchema, TFallback>>

```

Defined in:[packages/db/src/collection.ts:160](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L160)

Creates a new Collection instance with the given configuration
[Type Parameters](#type-parameters)

•**TExplicit**=unknown

The explicit type of items in the collection (highest priority)

•**TKey***extends*string|number=string|number

The type of the key for the collection

•**TUtils***extends*[UtilsRecord](/db/latest/docs/reference/type-aliases/utilsrecord)= {}

The utilities record type

•**TSchema***extends*StandardSchemaV1<unknown,unknown> =StandardSchemaV1<unknown,unknown>

The schema type for validation and type inference (second priority)

•**TFallback***extends*object=Record<string,unknown>

The fallback type if no explicit or schema type is provided
[Parameters](#parameters)[options](#options)

[CollectionConfig](/db/latest/docs/reference/interfaces/collectionconfig)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,TSchema,[ResolveInsertInput](/db/latest/docs/reference/type-aliases/resolveinsertinput)<TExplicit,TSchema,TFallback>> &object

Collection options with optional utilities
[Returns](#returns)

[Collection](/db/latest/docs/reference/interfaces/collection)<[ResolveType](/db/latest/docs/reference/type-aliases/resolvetype)<TExplicit,TSchema,TFallback>,TKey,TUtils,TSchema,[ResolveInsertInput](/db/latest/docs/reference/type-aliases/resolveinsertinput)<TExplicit,TSchema,TFallback>>

A new Collection with utilities exposed both at top level and under .utils
[Examples](#examples)ts

```
// Pattern 1: With operation handlers (direct collection calls)
const todos = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  schema,
  onInsert: async ({ transaction, collection }) => {
    // Send to API
    await api.createTodo(transaction.mutations[0].modified)
  },
  onUpdate: async ({ transaction, collection }) => {
    await api.updateTodo(transaction.mutations[0].modified)
  },
  onDelete: async ({ transaction, collection }) => {
    await api.deleteTodo(transaction.mutations[0].key)
  },
  sync: { sync: () => {} }
})

// Direct usage (handlers manage transactions)
const tx = todos.insert({ id: "1", text: "Buy milk", completed: false })
await tx.isPersisted.promise

```

```
// Pattern 1: With operation handlers (direct collection calls)
const todos = createCollection({
  id: "todos",
  getKey: (todo) => todo.id,
  schema,
  onInsert: async ({ transaction, collection }) => {
    // Send to API
    await api.createTodo(transaction.mutations[0].modified)
  },
  onUpdate: async ({ transaction, collection }) => {
    await api.updateTodo(transaction.mutations[0].modified)
  },
  onDelete: async ({ transaction, collection }) => {
    await api.deleteTodo(transaction.mutations[0].key)
  },
  sync: { sync: () => {} }
})

// Direct usage (handlers manage transactions)
const tx = todos.insert({ id: "1", text: "Buy milk", completed: false })
await tx.isPersisted.promise

```ts

```
// Pattern 2: Manual transaction management
const todos = createCollection({
  getKey: (todo) => todo.id,
  schema: todoSchema,
  sync: { sync: () => {} }
})

// Explicit transaction usage
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    // Handle all mutations in transaction
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  todos.insert({ id: "1", text: "Buy milk" })
  todos.update("2", draft => { draft.completed = true })
})

await tx.isPersisted.promise

```

```
// Pattern 2: Manual transaction management
const todos = createCollection({
  getKey: (todo) => todo.id,
  schema: todoSchema,
  sync: { sync: () => {} }
})

// Explicit transaction usage
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    // Handle all mutations in transaction
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  todos.insert({ id: "1", text: "Buy milk" })
  todos.update("2", draft => { draft.completed = true })
})

await tx.isPersisted.promise

```ts

```
// Using schema for type inference (preferred as it also gives you client side validation)
const todoSchema = z.object({
  id: z.string(),
  title: z.string(),
  completed: z.boolean()
})

const todos = createCollection({
  schema: todoSchema,
  getKey: (todo) => todo.id,
  sync: { sync: () => {} }
})

// Note: You must provide either an explicit type or a schema, but not both.

```

```
// Using schema for type inference (preferred as it also gives you client side validation)
const todoSchema = z.object({
  id: z.string(),
  title: z.string(),
  completed: z.boolean()
})

const todos = createCollection({
  schema: todoSchema,
  getKey: (todo) => todo.id,
  sync: { sync: () => {} }
})

// Note: You must provide either an explicit type or a schema, but not both.

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/createcollection.md)[Collection](/db/latest/docs/reference/interfaces/collection)[liveQueryCollectionOptions](/db/latest/docs/reference/functions/livequerycollectionoptions)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>