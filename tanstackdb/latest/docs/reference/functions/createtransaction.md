# createTransaction | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageFunction: createTransaction()Type ParametersParametersconfigReturnsExamples# createTransaction

Copy Markdown[Function: createTransaction()](#function-createtransaction)ts

```
function createTransaction<T>(config): Transaction<T>

```

```
function createTransaction<T>(config): Transaction<T>

```

Defined in:[packages/db/src/transactions.ts:74](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L74)

Creates a new transaction for grouping multiple collection operations
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>
[Parameters](#parameters)[config](#config)

[TransactionConfig](/db/latest/docs/reference/interfaces/transactionconfig)<T>

Transaction configuration with mutation function
[Returns](#returns)

[Transaction](/db/latest/docs/reference/classes/transaction)<T>

A new Transaction instance
[Examples](#examples)ts

```
// Basic transaction usage
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    // Send all mutations to API
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
  collection.update("2", draft => { draft.completed = true })
})

await tx.isPersisted.promise

```

```
// Basic transaction usage
const tx = createTransaction({
  mutationFn: async ({ transaction }) => {
    // Send all mutations to API
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
  collection.update("2", draft => { draft.completed = true })
})

await tx.isPersisted.promise

```ts

```
// Handle transaction errors
try {
  const tx = createTransaction({
    mutationFn: async () => { throw new Error("API failed") }
  })

  tx.mutate(() => {
    collection.insert({ id: "1", text: "New item" })
  })

  await tx.isPersisted.promise
} catch (error) {
  console.log('Transaction failed:', error)
}

```

```
// Handle transaction errors
try {
  const tx = createTransaction({
    mutationFn: async () => { throw new Error("API failed") }
  })

  tx.mutate(() => {
    collection.insert({ id: "1", text: "New item" })
  })

  await tx.isPersisted.promise
} catch (error) {
  console.log('Transaction failed:', error)
}

```ts

```
// Manual commit control
const tx = createTransaction({
  autoCommit: false,
  mutationFn: async () => {
    // API call
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

// Commit later
await tx.commit()

```

```
// Manual commit control
const tx = createTransaction({
  autoCommit: false,
  mutationFn: async () => {
    // API call
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

// Commit later
await tx.commit()

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/functions/createtransaction.md)[createOptimisticAction](/db/latest/docs/reference/functions/createoptimisticaction)[Electric DB Collection](/db/latest/docs/reference/electric-db-collection/index)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>