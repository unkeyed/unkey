# Transaction | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: Transaction<T>Type ParametersConstructorsnew Transaction()ParametersconfigReturnsPropertiesautoCommitcreatedAterror?errormessageidisPersistedmetadatamutationFnmutationssequenceNumberstateMethodsapplyMutations()ParametersmutationsReturnscommit()ReturnsExamplescompareCreatedAt()ParametersotherReturnsmutate()ParameterscallbackReturnsExamplesrollback()Parametersconfig?isSecondaryRollback?ReturnsExamplessetState()ParametersnewStateReturnstouchCollection()Returns# Transaction

Copy Markdown[Class: Transaction<T>](#class-transactiont)

Defined in:[packages/db/src/transactions.ts:116](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L116)
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>
[Constructors](#constructors)[new Transaction()](#new-transaction)ts

```
new Transaction<T>(config): Transaction<T>

```

```
new Transaction<T>(config): Transaction<T>

```

Defined in:[packages/db/src/transactions.ts:131](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L131)
[Parameters](#parameters)[config](#config)

[TransactionConfig](/db/latest/docs/reference/interfaces/transactionconfig)<T>
[Returns](#returns)

[Transaction](/db/latest/docs/reference/classes/transaction)<T>
[Properties](#properties)[autoCommit](#autocommit)ts

```
autoCommit: boolean;

```

```
autoCommit: boolean;

```

Defined in:[packages/db/src/transactions.ts:122](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L122)
[createdAt](#createdat)ts

```
createdAt: Date;

```

```
createdAt: Date;

```

Defined in:[packages/db/src/transactions.ts:123](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L123)
[error?](#error)ts

```
optional error: object;

```

```
optional error: object;

```

Defined in:[packages/db/src/transactions.ts:126](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L126)
[error](#error-1)ts

```
error: Error;

```

```
error: Error;

```[message](#message)ts

```
message: string;

```

```
message: string;

```[id](#id)ts

```
id: string;

```

```
id: string;

```

Defined in:[packages/db/src/transactions.ts:117](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L117)
[isPersisted](#ispersisted)ts

```
isPersisted: Deferred<Transaction<T>>;

```

```
isPersisted: Deferred<Transaction<T>>;

```

Defined in:[packages/db/src/transactions.ts:121](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L121)
[metadata](#metadata)ts

```
metadata: Record<string, unknown>;

```

```
metadata: Record<string, unknown>;

```

Defined in:[packages/db/src/transactions.ts:125](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L125)
[mutationFn](#mutationfn)ts

```
mutationFn: MutationFn<T>;

```

```
mutationFn: MutationFn<T>;

```

Defined in:[packages/db/src/transactions.ts:119](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L119)
[mutations](#mutations)ts

```
mutations: PendingMutation<T, OperationType, Collection<T, any, any, any, any>>[];

```

```
mutations: PendingMutation<T, OperationType, Collection<T, any, any, any, any>>[];

```

Defined in:[packages/db/src/transactions.ts:120](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L120)
[sequenceNumber](#sequencenumber)ts

```
sequenceNumber: number;

```

```
sequenceNumber: number;

```

Defined in:[packages/db/src/transactions.ts:124](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L124)
[state](#state)ts

```
state: TransactionState;

```

```
state: TransactionState;

```

Defined in:[packages/db/src/transactions.ts:118](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L118)
[Methods](#methods)[applyMutations()](#applymutations)ts

```
applyMutations(mutations): void

```

```
applyMutations(mutations): void

```

Defined in:[packages/db/src/transactions.ts:212](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L212)
[Parameters](#parameters-1)[mutations](#mutations-1)

[PendingMutation](/db/latest/docs/reference/interfaces/pendingmutation)<any,[OperationType](/db/latest/docs/reference/type-aliases/operationtype),[Collection](/db/latest/docs/reference/interfaces/collection)<any,any,any,any,any>>[]
[Returns](#returns-1)

void
[commit()](#commit)ts

```
commit(): Promise<Transaction<T>>

```

```
commit(): Promise<Transaction<T>>

```

Defined in:[packages/db/src/transactions.ts:349](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L349)

Commit the transaction and execute the mutation function
[Returns](#returns-2)

Promise<[Transaction](/db/latest/docs/reference/classes/transaction)<T>>

Promise that resolves to this transaction when complete
[Examples](#examples)ts

```
// Manual commit (when autoCommit is false)
const tx = createTransaction({
  autoCommit: false,
  mutationFn: async ({ transaction }) => {
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
})

await tx.commit() // Manually commit

```

```
// Manual commit (when autoCommit is false)
const tx = createTransaction({
  autoCommit: false,
  mutationFn: async ({ transaction }) => {
    await api.saveChanges(transaction.mutations)
  }
})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
})

await tx.commit() // Manually commit

```ts

```
// Handle commit errors
try {
  const tx = createTransaction({
    mutationFn: async () => { throw new Error("API failed") }
  })

  tx.mutate(() => {
    collection.insert({ id: "1", text: "Item" })
  })

  await tx.commit()
} catch (error) {
  console.log('Commit failed, transaction rolled back:', error)
}

```

```
// Handle commit errors
try {
  const tx = createTransaction({
    mutationFn: async () => { throw new Error("API failed") }
  })

  tx.mutate(() => {
    collection.insert({ id: "1", text: "Item" })
  })

  await tx.commit()
} catch (error) {
  console.log('Commit failed, transaction rolled back:', error)
}

```ts

```
// Check transaction state after commit
await tx.commit()
console.log(tx.state) // "completed" or "failed"

```

```
// Check transaction state after commit
await tx.commit()
console.log(tx.state) // "completed" or "failed"

```[compareCreatedAt()](#comparecreatedat)ts

```
compareCreatedAt(other): number

```

```
compareCreatedAt(other): number

```

Defined in:[packages/db/src/transactions.ts:395](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L395)

Compare two transactions by their createdAt time and sequence number in order
to sort them in the order they were created.
[Parameters](#parameters-2)[other](#other)

[Transaction](/db/latest/docs/reference/classes/transaction)<any>

The other transaction to compare to
[Returns](#returns-3)

number

-1 if this transaction was created before the other, 1 if it was created after, 0 if they were created at the same time
[mutate()](#mutate)ts

```
mutate(callback): Transaction<T>

```

```
mutate(callback): Transaction<T>

```

Defined in:[packages/db/src/transactions.ts:193](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L193)

Execute collection operations within this transaction
[Parameters](#parameters-3)[callback](#callback)

() =>void

Function containing collection operations to group together
[Returns](#returns-4)

[Transaction](/db/latest/docs/reference/classes/transaction)<T>

This transaction for chaining
[Examples](#examples-1)ts

```
// Group multiple operations
const tx = createTransaction({ mutationFn: async () => {
  // Send to API
}})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
  collection.update("2", draft => { draft.completed = true })
  collection.delete("3")
})

await tx.isPersisted.promise

```

```
// Group multiple operations
const tx = createTransaction({ mutationFn: async () => {
  // Send to API
}})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
  collection.update("2", draft => { draft.completed = true })
  collection.delete("3")
})

await tx.isPersisted.promise

```ts

```
// Handle mutate errors
try {
  tx.mutate(() => {
    collection.insert({ id: "invalid" }) // This might throw
  })
} catch (error) {
  console.log('Mutation failed:', error)
}

```

```
// Handle mutate errors
try {
  tx.mutate(() => {
    collection.insert({ id: "invalid" }) // This might throw
  })
} catch (error) {
  console.log('Mutation failed:', error)
}

```ts

```
// Manual commit control
const tx = createTransaction({ autoCommit: false, mutationFn: async () => {} })

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

// Commit later when ready
await tx.commit()

```

```
// Manual commit control
const tx = createTransaction({ autoCommit: false, mutationFn: async () => {} })

tx.mutate(() => {
  collection.insert({ id: "1", text: "Item" })
})

// Commit later when ready
await tx.commit()

```[rollback()](#rollback)ts

```
rollback(config?): Transaction<T>

```

```
rollback(config?): Transaction<T>

```

Defined in:[packages/db/src/transactions.ts:266](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L266)

Rollback the transaction and any conflicting transactions
[Parameters](#parameters-4)[config?](#config-1)

Configuration for rollback behavior
[isSecondaryRollback?](#issecondaryrollback)

boolean
[Returns](#returns-5)

[Transaction](/db/latest/docs/reference/classes/transaction)<T>

This transaction for chaining
[Examples](#examples-2)ts

```
// Manual rollback
const tx = createTransaction({ mutationFn: async () => {
  // Send to API
}})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
})

// Rollback if needed
if (shouldCancel) {
  tx.rollback()
}

```

```
// Manual rollback
const tx = createTransaction({ mutationFn: async () => {
  // Send to API
}})

tx.mutate(() => {
  collection.insert({ id: "1", text: "Buy milk" })
})

// Rollback if needed
if (shouldCancel) {
  tx.rollback()
}

```ts

```
// Handle rollback cascade (automatic)
const tx1 = createTransaction({ mutationFn: async () => {} })
const tx2 = createTransaction({ mutationFn: async () => {} })

tx1.mutate(() => collection.update("1", draft => { draft.value = "A" }))
tx2.mutate(() => collection.update("1", draft => { draft.value = "B" })) // Same item

tx1.rollback() // This will also rollback tx2 due to conflict

```

```
// Handle rollback cascade (automatic)
const tx1 = createTransaction({ mutationFn: async () => {} })
const tx2 = createTransaction({ mutationFn: async () => {} })

tx1.mutate(() => collection.update("1", draft => { draft.value = "A" }))
tx2.mutate(() => collection.update("1", draft => { draft.value = "B" })) // Same item

tx1.rollback() // This will also rollback tx2 due to conflict

```ts

```
// Handle rollback in error scenarios
try {
  await tx.isPersisted.promise
} catch (error) {
  console.log('Transaction was rolled back:', error)
  // Transaction automatically rolled back on mutation function failure
}

```

```
// Handle rollback in error scenarios
try {
  await tx.isPersisted.promise
} catch (error) {
  console.log('Transaction was rolled back:', error)
  // Transaction automatically rolled back on mutation function failure
}

```[setState()](#setstate)ts

```
setState(newState): void

```

```
setState(newState): void

```

Defined in:[packages/db/src/transactions.ts:146](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L146)
[Parameters](#parameters-5)[newState](#newstate)

[TransactionState](/db/latest/docs/reference/type-aliases/transactionstate)
[Returns](#returns-6)

void
[touchCollection()](#touchcollection)ts

```
touchCollection(): void

```

```
touchCollection(): void

```

Defined in:[packages/db/src/transactions.ts:294](https://github.com/TanStack/db/blob/main/packages/db/src/transactions.ts#L294)
[Returns](#returns-7)

void[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/transaction.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>