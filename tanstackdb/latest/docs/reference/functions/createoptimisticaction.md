# createOptimisticAction | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: createOptimisticAction()Type ParametersParametersoptionsReturnsParametersvariablesReturnsExample# createOptimisticAction

Copy Markdown[Function: createOptimisticAction()](#function-createoptimisticaction)ts

```
function createOptimisticAction<TVariables>(options): (variables) => Transaction

```

```
function createOptimisticAction<TVariables>(options): (variables) => Transaction

```

Defined in:[packages/db/src/optimistic-action.ts:41](https://github.com/TanStack/db/blob/main/packages/db/src/optimistic-action.ts#L41)

Creates an optimistic action function that applies local optimistic updates immediately
before executing the actual mutation on the server.

This pattern allows for responsive UI updates while the actual mutation is in progress.
The optimistic update is applied via theonMutatecallback, and the server mutation
is executed via themutationFn.
[Type Parameters](#type-parameters)

•**TVariables**=unknown

The type of variables that will be passed to the action function
[Parameters](#parameters)[options](#options)

[CreateOptimisticActionsOptions](/db/latest/docs/reference/interfaces/createoptimisticactionsoptions)<TVariables>

Configuration options for the optimistic action
[Returns](#returns)

Function

A function that accepts variables of type TVariables and returns a Transaction
[Parameters](#parameters-1)[variables](#variables)

TVariables
[Returns](#returns-1)

[Transaction](/db/latest/docs/reference/classes/transaction)
[Example](#example)ts

```
const addTodo = createOptimisticAction<string>({
  onMutate: (text) => {
    // Instantly applies local optimistic state
    todoCollection.insert({
      id: uuid(),
      text,
      completed: false
    })
  },
  mutationFn: async (text, params) => {
    // Persist the todo to your backend
    const response = await fetch('/api/todos', {
      method: 'POST',
      body: JSON.stringify({ text, completed: false }),
    })
    return response.json()
  }
})

// Usage
const transaction = addTodo('New Todo Item')

```

```
const addTodo = createOptimisticAction<string>({
  onMutate: (text) => {
    // Instantly applies local optimistic state
    todoCollection.insert({
      id: uuid(),
      text,
      completed: false
    })
  },
  mutationFn: async (text, params) => {
    // Persist the todo to your backend
    const response = await fetch('/api/todos', {
      method: 'POST',
      body: JSON.stringify({ text, completed: false }),
    })
    return response.json()
  }
})

// Usage
const transaction = addTodo('New Todo Item')

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/createoptimisticaction.md)[createLiveQueryCollection](/db/latest/docs/reference/functions/createlivequerycollection)[createTransaction](/db/latest/docs/reference/functions/createtransaction)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>