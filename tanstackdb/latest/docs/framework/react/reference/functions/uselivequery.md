# useLiveQuery | TanStack DB React Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: useLiveQuery()Call SignatureType ParametersParametersqueryFndeps?ReturnscollectiondataisCleanedUpisErrorisIdleisLoadingisReadystatestatusExamplesCall SignatureType ParametersParametersconfigdeps?ReturnscollectiondataisCleanedUpisErrorisIdleisLoadingisReadystatestatusExamplesCall SignatureType ParametersParametersliveQueryCollectionReturnscollectiondataisCleanedUpisErrorisIdleisLoadingisReadystatestatusExamples# useLiveQuery

Copy Markdown[Function: useLiveQuery()](#function-uselivequery)[Call Signature](#call-signature)ts

```
function useLiveQuery<TContext>(queryFn, deps?): object

```

```
function useLiveQuery<TContext>(queryFn, deps?): object

```

Defined in:[useLiveQuery.ts:64](https://github.com/TanStack/db/blob/main/packages/react-db/src/useLiveQuery.ts#L64)

Create a live query using a query function
[Type Parameters](#type-parameters)

•**TContext***extends*Context
[Parameters](#parameters)[queryFn](#queryfn)

(q) =>QueryBuilder<TContext>

Query function that defines what data to fetch
[deps?](#deps)

unknown[]

Array of dependencies that trigger query re-execution when changed
[Returns](#returns)

object

Object with reactive data, state, and status information
[collection](#collection)ts

```
collection: Collection<{ [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }, string | number, {}>;

```

```
collection: Collection<{ [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }, string | number, {}>;

```[data](#data)ts

```
data: { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }[];

```

```
data: { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }[];

```[isCleanedUp](#iscleanedup)ts

```
isCleanedUp: boolean;

```

```
isCleanedUp: boolean;

```[isError](#iserror)ts

```
isError: boolean;

```

```
isError: boolean;

```[isIdle](#isidle)ts

```
isIdle: boolean;

```

```
isIdle: boolean;

```[isLoading](#isloading)ts

```
isLoading: boolean;

```

```
isLoading: boolean;

```[isReady](#isready)ts

```
isReady: boolean;

```

```
isReady: boolean;

```[state](#state)ts

```
state: Map<string | number, { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }>;

```

```
state: Map<string | number, { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }>;

```[status](#status)ts

```
status: CollectionStatus;

```

```
status: CollectionStatus;

```[Examples](#examples)ts

```
// Basic query with object syntax
const { data, isLoading } = useLiveQuery((q) =>
  q.from({ todos: todosCollection })
   .where(({ todos }) => eq(todos.completed, false))
   .select(({ todos }) => ({ id: todos.id, text: todos.text }))
)

```

```
// Basic query with object syntax
const { data, isLoading } = useLiveQuery((q) =>
  q.from({ todos: todosCollection })
   .where(({ todos }) => eq(todos.completed, false))
   .select(({ todos }) => ({ id: todos.id, text: todos.text }))
)

```ts

```
// With dependencies that trigger re-execution
const { data, state } = useLiveQuery(
  (q) => q.from({ todos: todosCollection })
         .where(({ todos }) => gt(todos.priority, minPriority)),
  [minPriority] // Re-run when minPriority changes
)

```

```
// With dependencies that trigger re-execution
const { data, state } = useLiveQuery(
  (q) => q.from({ todos: todosCollection })
         .where(({ todos }) => gt(todos.priority, minPriority)),
  [minPriority] // Re-run when minPriority changes
)

```ts

```
// Join pattern
const { data } = useLiveQuery((q) =>
  q.from({ issues: issueCollection })
   .join({ persons: personCollection }, ({ issues, persons }) =>
     eq(issues.userId, persons.id)
   )
   .select(({ issues, persons }) => ({
     id: issues.id,
     title: issues.title,
     userName: persons.name
   }))
)

```

```
// Join pattern
const { data } = useLiveQuery((q) =>
  q.from({ issues: issueCollection })
   .join({ persons: personCollection }, ({ issues, persons }) =>
     eq(issues.userId, persons.id)
   )
   .select(({ issues, persons }) => ({
     id: issues.id,
     title: issues.title,
     userName: persons.name
   }))
)

```ts

```
// Handle loading and error states
const { data, isLoading, isError, status } = useLiveQuery((q) =>
  q.from({ todos: todoCollection })
)

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Error: {status}</div>

return (
  <ul>
    {data.map(todo => <li key={todo.id}>{todo.text}</li>)}
  </ul>
)

```

```
// Handle loading and error states
const { data, isLoading, isError, status } = useLiveQuery((q) =>
  q.from({ todos: todoCollection })
)

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Error: {status}</div>

return (
  <ul>
    {data.map(todo => <li key={todo.id}>{todo.text}</li>)}
  </ul>
)

```[Call Signature](#call-signature-1)ts

```
function useLiveQuery<TContext>(config, deps?): object

```

```
function useLiveQuery<TContext>(config, deps?): object

```

Defined in:[useLiveQuery.ts:113](https://github.com/TanStack/db/blob/main/packages/react-db/src/useLiveQuery.ts#L113)

Create a live query using configuration object
[Type Parameters](#type-parameters-1)

•**TContext***extends*Context
[Parameters](#parameters-1)[config](#config)

LiveQueryCollectionConfig<TContext>

Configuration object with query and options
[deps?](#deps-1)

unknown[]

Array of dependencies that trigger query re-execution when changed
[Returns](#returns-1)

object

Object with reactive data, state, and status information
[collection](#collection-1)ts

```
collection: Collection<{ [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }, string | number, {}>;

```

```
collection: Collection<{ [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }, string | number, {}>;

```[data](#data-1)ts

```
data: { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }[];

```

```
data: { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }[];

```[isCleanedUp](#iscleanedup-1)ts

```
isCleanedUp: boolean;

```

```
isCleanedUp: boolean;

```[isError](#iserror-1)ts

```
isError: boolean;

```

```
isError: boolean;

```[isIdle](#isidle-1)ts

```
isIdle: boolean;

```

```
isIdle: boolean;

```[isLoading](#isloading-1)ts

```
isLoading: boolean;

```

```
isLoading: boolean;

```[isReady](#isready-1)ts

```
isReady: boolean;

```

```
isReady: boolean;

```[state](#state-1)ts

```
state: Map<string | number, { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }>;

```

```
state: Map<string | number, { [K in string | number | symbol]: (TContext["result"] extends object ? any[any] : TContext["hasJoins"] extends true ? TContext["schema"] : TContext["schema"][TContext["fromSourceName"]])[K] }>;

```[status](#status-1)ts

```
status: CollectionStatus;

```

```
status: CollectionStatus;

```[Examples](#examples-1)ts

```
// Basic config object usage
const { data, status } = useLiveQuery({
  query: (q) => q.from({ todos: todosCollection }),
  gcTime: 60000
})

```

```
// Basic config object usage
const { data, status } = useLiveQuery({
  query: (q) => q.from({ todos: todosCollection }),
  gcTime: 60000
})

```ts

```
// With query builder and options
const queryBuilder = new Query()
  .from({ persons: collection })
  .where(({ persons }) => gt(persons.age, 30))
  .select(({ persons }) => ({ id: persons.id, name: persons.name }))

const { data, isReady } = useLiveQuery({ query: queryBuilder })

```

```
// With query builder and options
const queryBuilder = new Query()
  .from({ persons: collection })
  .where(({ persons }) => gt(persons.age, 30))
  .select(({ persons }) => ({ id: persons.id, name: persons.name }))

const { data, isReady } = useLiveQuery({ query: queryBuilder })

```ts

```
// Handle all states uniformly
const { data, isLoading, isReady, isError } = useLiveQuery({
  query: (q) => q.from({ items: itemCollection })
})

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Something went wrong</div>
if (!isReady) return <div>Preparing...</div>

return <div>{data.length} items loaded</div>

```

```
// Handle all states uniformly
const { data, isLoading, isReady, isError } = useLiveQuery({
  query: (q) => q.from({ items: itemCollection })
})

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Something went wrong</div>
if (!isReady) return <div>Preparing...</div>

return <div>{data.length} items loaded</div>

```[Call Signature](#call-signature-2)ts

```
function useLiveQuery<TResult, TKey, TUtils>(liveQueryCollection): object

```

```
function useLiveQuery<TResult, TKey, TUtils>(liveQueryCollection): object

```

Defined in:[useLiveQuery.ts:158](https://github.com/TanStack/db/blob/main/packages/react-db/src/useLiveQuery.ts#L158)

Subscribe to an existing live query collection
[Type Parameters](#type-parameters-2)

•**TResult***extends*object

•**TKey***extends*string|number

•**TUtils***extends*Record<string,any>
[Parameters](#parameters-2)[liveQueryCollection](#livequerycollection)

Collection<TResult,TKey,TUtils>

Pre-created live query collection to subscribe to
[Returns](#returns-2)

object

Object with reactive data, state, and status information
[collection](#collection-2)ts

```
collection: Collection<TResult, TKey, TUtils>;

```

```
collection: Collection<TResult, TKey, TUtils>;

```[data](#data-2)ts

```
data: TResult[];

```

```
data: TResult[];

```[isCleanedUp](#iscleanedup-2)ts

```
isCleanedUp: boolean;

```

```
isCleanedUp: boolean;

```[isError](#iserror-2)ts

```
isError: boolean;

```

```
isError: boolean;

```[isIdle](#isidle-2)ts

```
isIdle: boolean;

```

```
isIdle: boolean;

```[isLoading](#isloading-2)ts

```
isLoading: boolean;

```

```
isLoading: boolean;

```[isReady](#isready-2)ts

```
isReady: boolean;

```

```
isReady: boolean;

```[state](#state-2)ts

```
state: Map<TKey, TResult>;

```

```
state: Map<TKey, TResult>;

```[status](#status-2)ts

```
status: CollectionStatus;

```

```
status: CollectionStatus;

```[Examples](#examples-2)ts

```
// Using pre-created live query collection
const myLiveQuery = createLiveQueryCollection((q) =>
  q.from({ todos: todosCollection }).where(({ todos }) => eq(todos.active, true))
)
const { data, collection } = useLiveQuery(myLiveQuery)

```

```
// Using pre-created live query collection
const myLiveQuery = createLiveQueryCollection((q) =>
  q.from({ todos: todosCollection }).where(({ todos }) => eq(todos.active, true))
)
const { data, collection } = useLiveQuery(myLiveQuery)

```ts

```
// Access collection methods directly
const { data, collection, isReady } = useLiveQuery(existingCollection)

// Use collection for mutations
const handleToggle = (id) => {
  collection.update(id, draft => { draft.completed = !draft.completed })
}

```

```
// Access collection methods directly
const { data, collection, isReady } = useLiveQuery(existingCollection)

// Use collection for mutations
const handleToggle = (id) => {
  collection.update(id, draft => { draft.completed = !draft.completed })
}

```ts

```
// Handle states consistently
const { data, isLoading, isError } = useLiveQuery(sharedCollection)

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Error loading data</div>

return <div>{data.map(item => <Item key={item.id} {...item} />)}</div>

```

```
// Handle states consistently
const { data, isLoading, isError } = useLiveQuery(sharedCollection)

if (isLoading) return <div>Loading...</div>
if (isError) return <div>Error loading data</div>

return <div>{data.map(item => <Item key={item.id} {...item} />)}</div>

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/framework/react/reference/functions/uselivequery.md)[React Hooks](/db/latest/docs/framework/react/reference/index)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>