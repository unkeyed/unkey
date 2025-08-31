# Collection | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageInterface: Collection<T, TKey, TUtils, TSchema, TInsertInput>ExtendsType ParametersPropertiesconfigInherited fromidInherited fromoptimisticDeletesInherited fromoptimisticUpsertsInherited frompendingSyncedTransactionsInherited fromsyncedDataInherited fromsyncedMetadataInherited fromtransactionsInherited fromutilsOverridesAccessorsindexesGet SignatureReturnsInherited fromsizeGet SignatureReturnsInherited fromstateGet SignatureExampleReturnsInherited fromstatusGet SignatureReturnsInherited fromtoArrayGet SignatureReturnsInherited fromMethods[iterator]()ReturnsInherited fromcleanup()ReturnsInherited fromcommitPendingTransactions()ReturnsInherited fromcreateIndex()Type ParametersParametersindexCallbackconfigReturnsExampleInherited fromcurrentStateAsChanges()ParametersoptionsReturnsExampleInherited fromdelete()Parameterskeysconfig?ReturnsExamplesInherited fromentries()ReturnsInherited fromforEach()ParameterscallbackfnReturnsInherited fromgenerateGlobalKey()ParameterskeyitemReturnsInherited fromget()ParameterskeyReturnsInherited fromgetKeyFromItem()ParametersitemReturnsInherited fromhas()ParameterskeyReturnsInherited frominsert()Parametersdataconfig?ReturnsThrowsExamplesInherited fromisReady()ReturnsExampleInherited fromkeys()ReturnsInherited frommap()Type ParametersParameterscallbackfnReturnsInherited fromonFirstReady()ParameterscallbackReturnsExampleInherited fromonTransactionStateChange()ReturnsInherited frompreload()ReturnsInherited fromstartSyncImmediate()ReturnsInherited fromstateWhenReady()ReturnsInherited fromsubscribeChanges()ParameterscallbackoptionsReturnsReturnsExamplesInherited fromsubscribeChangesKey()Parameterskeylistener__namedParametersincludeInitialState?ReturnsReturnsInherited fromtoArrayWhenReady()ReturnsInherited fromupdate()Call SignatureType ParametersParameterskeycallbackReturnsThrowsExamplesInherited fromCall SignatureType ParametersParameterskeysconfigcallbackReturnsThrowsExamplesInherited fromCall SignatureType ParametersParametersidcallbackReturnsThrowsExamplesInherited fromCall SignatureType ParametersParametersidconfigcallbackReturnsThrowsExamplesInherited fromvalues()ReturnsInherited from# Collection

Copy Markdown[Interface: Collection<T, TKey, TUtils, TSchema, TInsertInput>](#interface-collectiont-tkey-tutils-tschema-tinsertinput)

Defined in:[packages/db/src/collection.ts:77](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L77)

Enhanced Collection interface that includes both data type T and utilities TUtils
[Extends](#extends)

- [CollectionImpl](/db/latest/docs/reference/classes/collectionimpl)<T,TKey,TUtils,TSchema,TInsertInput>
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>

The type of items in the collection

•**TKey***extends*string|number=string|number

The type of the key for the collection

•**TUtils***extends*[UtilsRecord](/db/latest/docs/reference/type-aliases/utilsrecord)= {}

The utilities record type

•**TSchema***extends*StandardSchemaV1=StandardSchemaV1

•**TInsertInput***extends*object=T

The type for insert operations (can be different from T for schemas with defaults)
[Properties](#properties)[config](#config)ts

```
config: CollectionConfig<T, TKey, TSchema, TInsertInput>;

```

```
config: CollectionConfig<T, TKey, TSchema, TInsertInput>;

```

Defined in:[packages/db/src/collection.ts:211](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L211)
[Inherited from](#inherited-from)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[config](/db/latest/docs/reference/classes/CollectionImpl#config-1)
[id](#id)ts

```
id: string;

```

```
id: string;

```

Defined in:[packages/db/src/collection.ts:331](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L331)
[Inherited from](#inherited-from-1)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[id](/db/latest/docs/reference/classes/CollectionImpl#id)
[optimisticDeletes](#optimisticdeletes)ts

```
optimisticDeletes: Set<TKey>;

```

```
optimisticDeletes: Set<TKey>;

```

Defined in:[packages/db/src/collection.ts:221](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L221)
[Inherited from](#inherited-from-2)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[optimisticDeletes](/db/latest/docs/reference/classes/CollectionImpl#optimisticdeletes)
[optimisticUpserts](#optimisticupserts)ts

```
optimisticUpserts: Map<TKey, T>;

```

```
optimisticUpserts: Map<TKey, T>;

```

Defined in:[packages/db/src/collection.ts:220](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L220)
[Inherited from](#inherited-from-3)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[optimisticUpserts](/db/latest/docs/reference/classes/CollectionImpl#optimisticupserts)
[pendingSyncedTransactions](#pendingsyncedtransactions)ts

```
pendingSyncedTransactions: PendingSyncedTransaction<T>[] = [];

```

```
pendingSyncedTransactions: PendingSyncedTransaction<T>[] = [];

```

Defined in:[packages/db/src/collection.ts:215](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L215)
[Inherited from](#inherited-from-4)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[pendingSyncedTransactions](/db/latest/docs/reference/classes/CollectionImpl#pendingsyncedtransactions)
[syncedData](#synceddata)ts

```
syncedData: 
  | Map<TKey, T>
| SortedMap<TKey, T>;

```

```
syncedData: 
  | Map<TKey, T>
| SortedMap<TKey, T>;

```

Defined in:[packages/db/src/collection.ts:216](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L216)
[Inherited from](#inherited-from-5)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[syncedData](/db/latest/docs/reference/classes/CollectionImpl#synceddata)
[syncedMetadata](#syncedmetadata)ts

```
syncedMetadata: Map<TKey, unknown>;

```

```
syncedMetadata: Map<TKey, unknown>;

```

Defined in:[packages/db/src/collection.ts:217](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L217)
[Inherited from](#inherited-from-6)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[syncedMetadata](/db/latest/docs/reference/classes/CollectionImpl#syncedmetadata)
[transactions](#transactions)ts

```
transactions: SortedMap<string, Transaction<any>>;

```

```
transactions: SortedMap<string, Transaction<any>>;

```

Defined in:[packages/db/src/collection.ts:214](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L214)
[Inherited from](#inherited-from-7)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[transactions](/db/latest/docs/reference/classes/CollectionImpl#transactions)
[utils](#utils)ts

```
readonly utils: TUtils;

```

```
readonly utils: TUtils;

```

Defined in:[packages/db/src/collection.ts:84](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L84)
[Overrides](#overrides)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[utils](/db/latest/docs/reference/classes/CollectionImpl#utils)
[Accessors](#accessors)[indexes](#indexes)[Get Signature](#get-signature)ts

```
get indexes(): Map<number, BaseIndex<TKey>>

```

```
get indexes(): Map<number, BaseIndex<TKey>>

```

Defined in:[packages/db/src/collection.ts:1439](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1439)

Get resolved indexes for query optimization
[Returns](#returns)

Map<number,[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>>
[Inherited from](#inherited-from-8)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[indexes](/db/latest/docs/reference/classes/CollectionImpl#indexes)
[size](#size)[Get Signature](#get-signature-1)ts

```
get size(): number

```

```
get size(): number

```

Defined in:[packages/db/src/collection.ts:995](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L995)

Get the current size of the collection (cached)
[Returns](#returns-1)

number
[Inherited from](#inherited-from-9)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[size](/db/latest/docs/reference/classes/CollectionImpl#size)
[state](#state)[Get Signature](#get-signature-2)ts

```
get state(): Map<TKey, T>

```

```
get state(): Map<TKey, T>

```

Defined in:[packages/db/src/collection.ts:2038](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2038)

Gets the current state of the collection as a Map
[Example](#example)ts

```
const itemsMap = collection.state
console.log(`Collection has ${itemsMap.size} items`)

for (const [key, item] of itemsMap) {
  console.log(`${key}: ${item.title}`)
}

// Check if specific item exists
if (itemsMap.has("todo-1")) {
  console.log("Todo 1 exists:", itemsMap.get("todo-1"))
}

```

```
const itemsMap = collection.state
console.log(`Collection has ${itemsMap.size} items`)

for (const [key, item] of itemsMap) {
  console.log(`${key}: ${item.title}`)
}

// Check if specific item exists
if (itemsMap.has("todo-1")) {
  console.log("Todo 1 exists:", itemsMap.get("todo-1"))
}

```[Returns](#returns-2)

Map<TKey,T>

Map containing all items in the collection, with keys as identifiers
[Inherited from](#inherited-from-10)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[state](/db/latest/docs/reference/classes/CollectionImpl#state)
[status](#status)[Get Signature](#get-signature-3)ts

```
get status(): CollectionStatus

```

```
get status(): CollectionStatus

```

Defined in:[packages/db/src/collection.ts:336](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L336)

Gets the current status of the collection
[Returns](#returns-3)

[CollectionStatus](/db/latest/docs/reference/type-aliases/collectionstatus)
[Inherited from](#inherited-from-11)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[status](/db/latest/docs/reference/classes/CollectionImpl#status)
[toArray](#toarray)[Get Signature](#get-signature-4)ts

```
get toArray(): T[]

```

```
get toArray(): T[]

```

Defined in:[packages/db/src/collection.ts:2071](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2071)

Gets the current state of the collection as an Array
[Returns](#returns-4)

T[]

An Array containing all items in the collection
[Inherited from](#inherited-from-12)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[toArray](/db/latest/docs/reference/classes/CollectionImpl#toarray)
[Methods](#methods)[[iterator]()](#iterator)ts

```
iterator: IterableIterator<[TKey, T]>

```

```
iterator: IterableIterator<[TKey, T]>

```

Defined in:[packages/db/src/collection.ts:1046](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1046)

Get all entries (virtual derived state)
[Returns](#returns-5)

IterableIterator<[TKey,T]>
[Inherited from](#inherited-from-13)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[[iterator]](/db/latest/docs/reference/classes/CollectionImpl#iterator)
[cleanup()](#cleanup)ts

```
cleanup(): Promise<void>

```

```
cleanup(): Promise<void>

```

Defined in:[packages/db/src/collection.ts:583](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L583)

Clean up the collection by stopping sync and clearing data
This can be called manually or automatically by garbage collection
[Returns](#returns-6)

Promise<void>
[Inherited from](#inherited-from-14)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[cleanup](/db/latest/docs/reference/classes/CollectionImpl#cleanup)
[commitPendingTransactions()](#commitpendingtransactions)ts

```
commitPendingTransactions(): void

```

```
commitPendingTransactions(): void

```

Defined in:[packages/db/src/collection.ts:1082](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1082)

Attempts to commit pending synced transactions if there are no active transactions
This method processes operations from pending transactions and applies them to the synced data
[Returns](#returns-7)

void
[Inherited from](#inherited-from-15)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[commitPendingTransactions](/db/latest/docs/reference/classes/CollectionImpl#commitpendingtransactions)
[createIndex()](#createindex)ts

```
createIndex<TResolver>(indexCallback, config): IndexProxy<TKey>

```

```
createIndex<TResolver>(indexCallback, config): IndexProxy<TKey>

```

Defined in:[packages/db/src/collection.ts:1344](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1344)

Creates an index on a collection for faster queries.
Indexes significantly improve query performance by allowing binary search
and range queries instead of full scans.
[Type Parameters](#type-parameters-1)

•**TResolver***extends*[IndexResolver](/db/latest/docs/reference/type-aliases/indexresolver)<TKey> =*typeof*[BTreeIndex](/db/latest/docs/reference/classes/btreeindex)

The type of the index resolver (constructor or async loader)
[Parameters](#parameters)[indexCallback](#indexcallback)

(row) =>any

Function that extracts the indexed value from each item
[config](#config-1)

[IndexOptions](/db/latest/docs/reference/interfaces/indexoptions)<TResolver> ={}

Configuration including index type and type-specific options
[Returns](#returns-8)

[IndexProxy](/db/latest/docs/reference/classes/indexproxy)<TKey>

An index proxy that provides access to the index when ready
[Example](#example-1)ts

```
// Create a default B+ tree index
const ageIndex = collection.createIndex((row) => row.age)

// Create a ordered index with custom options
const ageIndex = collection.createIndex((row) => row.age, {
  indexType: BTreeIndex,
  options: { compareFn: customComparator },
  name: 'age_btree'
})

// Create an async-loaded index
const textIndex = collection.createIndex((row) => row.content, {
  indexType: async () => {
    const { FullTextIndex } = await import('./indexes/fulltext.js')
    return FullTextIndex
  },
  options: { language: 'en' }
})

```

```
// Create a default B+ tree index
const ageIndex = collection.createIndex((row) => row.age)

// Create a ordered index with custom options
const ageIndex = collection.createIndex((row) => row.age, {
  indexType: BTreeIndex,
  options: { compareFn: customComparator },
  name: 'age_btree'
})

// Create an async-loaded index
const textIndex = collection.createIndex((row) => row.content, {
  indexType: async () => {
    const { FullTextIndex } = await import('./indexes/fulltext.js')
    return FullTextIndex
  },
  options: { language: 'en' }
})

```[Inherited from](#inherited-from-16)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[createIndex](/db/latest/docs/reference/classes/CollectionImpl#createindex)
[currentStateAsChanges()](#currentstateaschanges)ts

```
currentStateAsChanges(options): ChangeMessage<T, string | number>[]

```

```
currentStateAsChanges(options): ChangeMessage<T, string | number>[]

```

Defined in:[packages/db/src/collection.ts:2113](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2113)

Returns the current state of the collection as an array of changes
[Parameters](#parameters-1)[options](#options)

[CurrentStateAsChangesOptions](/db/latest/docs/reference/interfaces/currentstateaschangesoptions)<T> ={}

Options including optional where filter
[Returns](#returns-9)

[ChangeMessage](/db/latest/docs/reference/interfaces/changemessage)<T,string|number>[]

An array of changes
[Example](#example-2)ts

```
// Get all items as changes
const allChanges = collection.currentStateAsChanges()

// Get only items matching a condition
const activeChanges = collection.currentStateAsChanges({
  where: (row) => row.status === 'active'
})

// Get only items using a pre-compiled expression
const activeChanges = collection.currentStateAsChanges({
  whereExpression: eq(row.status, 'active')
})

```

```
// Get all items as changes
const allChanges = collection.currentStateAsChanges()

// Get only items matching a condition
const activeChanges = collection.currentStateAsChanges({
  where: (row) => row.status === 'active'
})

// Get only items using a pre-compiled expression
const activeChanges = collection.currentStateAsChanges({
  whereExpression: eq(row.status, 'active')
})

```[Inherited from](#inherited-from-17)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[currentStateAsChanges](/db/latest/docs/reference/classes/CollectionImpl#currentstateaschanges)
[delete()](#delete)ts

```
delete(keys, config?): Transaction<any>

```

```
delete(keys, config?): Transaction<any>

```

Defined in:[packages/db/src/collection.ts:1939](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1939)

Deletes one or more items from the collection
[Parameters](#parameters-2)[keys](#keys)

Single key or array of keys to delete

TKey|TKey[]
[config?](#config-2)

[OperationConfig](/db/latest/docs/reference/interfaces/operationconfig)

Optional configuration including metadata
[Returns](#returns-10)

[Transaction](/db/latest/docs/reference/classes/transaction)<any>

A Transaction object representing the delete operation(s)
[Examples](#examples)ts

```
// Delete a single item
const tx = collection.delete("todo-1")
await tx.isPersisted.promise

```

```
// Delete a single item
const tx = collection.delete("todo-1")
await tx.isPersisted.promise

```ts

```
// Delete multiple items
const tx = collection.delete(["todo-1", "todo-2"])
await tx.isPersisted.promise

```

```
// Delete multiple items
const tx = collection.delete(["todo-1", "todo-2"])
await tx.isPersisted.promise

```ts

```
// Delete with metadata
const tx = collection.delete("todo-1", { metadata: { reason: "completed" } })
await tx.isPersisted.promise

```

```
// Delete with metadata
const tx = collection.delete("todo-1", { metadata: { reason: "completed" } })
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.delete("item-1")
  await tx.isPersisted.promise
  console.log('Delete successful')
} catch (error) {
  console.log('Delete failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.delete("item-1")
  await tx.isPersisted.promise
  console.log('Delete successful')
} catch (error) {
  console.log('Delete failed:', error)
}

```[Inherited from](#inherited-from-18)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[delete](/db/latest/docs/reference/classes/CollectionImpl#delete)
[entries()](#entries)ts

```
entries(): IterableIterator<[TKey, T]>

```

```
entries(): IterableIterator<[TKey, T]>

```

Defined in:[packages/db/src/collection.ts:1034](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1034)

Get all entries (virtual derived state)
[Returns](#returns-11)

IterableIterator<[TKey,T]>
[Inherited from](#inherited-from-19)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[entries](/db/latest/docs/reference/classes/CollectionImpl#entries)
[forEach()](#foreach)ts

```
forEach(callbackfn): void

```

```
forEach(callbackfn): void

```

Defined in:[packages/db/src/collection.ts:1055](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1055)

Execute a callback for each entry in the collection
[Parameters](#parameters-3)[callbackfn](#callbackfn)

(value,key,index) =>void
[Returns](#returns-12)

void
[Inherited from](#inherited-from-20)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[forEach](/db/latest/docs/reference/classes/CollectionImpl#foreach)
[generateGlobalKey()](#generateglobalkey)ts

```
generateGlobalKey(key, item): string

```

```
generateGlobalKey(key, item): string

```

Defined in:[packages/db/src/collection.ts:1306](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1306)
[Parameters](#parameters-4)[key](#key)

any
[item](#item)

any
[Returns](#returns-13)

string
[Inherited from](#inherited-from-21)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[generateGlobalKey](/db/latest/docs/reference/classes/CollectionImpl#generateglobalkey)
[get()](#get)ts

```
get(key): undefined | T

```

```
get(key): undefined | T

```

Defined in:[packages/db/src/collection.ts:959](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L959)

Get the current value for a key (virtual derived state)
[Parameters](#parameters-5)[key](#key-1)

TKey
[Returns](#returns-14)

undefined|T
[Inherited from](#inherited-from-22)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[get](/db/latest/docs/reference/classes/CollectionImpl#get)
[getKeyFromItem()](#getkeyfromitem)ts

```
getKeyFromItem(item): TKey

```

```
getKeyFromItem(item): TKey

```

Defined in:[packages/db/src/collection.ts:1302](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1302)
[Parameters](#parameters-6)[item](#item-1)

T
[Returns](#returns-15)

TKey
[Inherited from](#inherited-from-23)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[getKeyFromItem](/db/latest/docs/reference/classes/CollectionImpl#getkeyfromitem)
[has()](#has)ts

```
has(key): boolean

```

```
has(key): boolean

```

Defined in:[packages/db/src/collection.ts:977](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L977)

Check if a key exists in the collection (virtual derived state)
[Parameters](#parameters-7)[key](#key-2)

TKey
[Returns](#returns-16)

boolean
[Inherited from](#inherited-from-24)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[has](/db/latest/docs/reference/classes/CollectionImpl#has)
[insert()](#insert)ts

```
insert(data, config?): 
  | Transaction<Record<string, unknown>>
| Transaction<T>

```

```
insert(data, config?): 
  | Transaction<Record<string, unknown>>
| Transaction<T>

```

Defined in:[packages/db/src/collection.ts:1594](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1594)

Inserts one or more items into the collection
[Parameters](#parameters-8)[data](#data)

TInsertInput|TInsertInput[]
[config?](#config-3)

[InsertConfig](/db/latest/docs/reference/interfaces/insertconfig)

Optional configuration including metadata
[Returns](#returns-17)

|[Transaction](/db/latest/docs/reference/classes/transaction)<Record<string,unknown>>
  |[Transaction](/db/latest/docs/reference/classes/transaction)<T>

A Transaction object representing the insert operation(s)
[Throws](#throws)

If the data fails schema validation
[Examples](#examples-1)ts

```
// Insert a single todo (requires onInsert handler)
const tx = collection.insert({ id: "1", text: "Buy milk", completed: false })
await tx.isPersisted.promise

```

```
// Insert a single todo (requires onInsert handler)
const tx = collection.insert({ id: "1", text: "Buy milk", completed: false })
await tx.isPersisted.promise

```ts

```
// Insert multiple todos at once
const tx = collection.insert([
  { id: "1", text: "Buy milk", completed: false },
  { id: "2", text: "Walk dog", completed: true }
])
await tx.isPersisted.promise

```

```
// Insert multiple todos at once
const tx = collection.insert([
  { id: "1", text: "Buy milk", completed: false },
  { id: "2", text: "Walk dog", completed: true }
])
await tx.isPersisted.promise

```ts

```
// Insert with metadata
const tx = collection.insert({ id: "1", text: "Buy groceries" },
  { metadata: { source: "mobile-app" } }
)
await tx.isPersisted.promise

```

```
// Insert with metadata
const tx = collection.insert({ id: "1", text: "Buy groceries" },
  { metadata: { source: "mobile-app" } }
)
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.insert({ id: "1", text: "New item" })
  await tx.isPersisted.promise
  console.log('Insert successful')
} catch (error) {
  console.log('Insert failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.insert({ id: "1", text: "New item" })
  await tx.isPersisted.promise
  console.log('Insert successful')
} catch (error) {
  console.log('Insert failed:', error)
}

```[Inherited from](#inherited-from-25)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[insert](/db/latest/docs/reference/classes/CollectionImpl#insert)
[isReady()](#isready)ts

```
isReady(): boolean

```

```
isReady(): boolean

```

Defined in:[packages/db/src/collection.ts:294](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L294)

Check if the collection is ready for use
Returns true if the collection has been marked as ready by its sync implementation
[Returns](#returns-18)

boolean

true if the collection is ready, false otherwise
[Example](#example-3)ts

```
if (collection.isReady()) {
  console.log('Collection is ready, data is available')
  // Safe to access collection.state
} else {
  console.log('Collection is still loading')
}

```

```
if (collection.isReady()) {
  console.log('Collection is ready, data is available')
  // Safe to access collection.state
} else {
  console.log('Collection is still loading')
}

```[Inherited from](#inherited-from-26)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[isReady](/db/latest/docs/reference/classes/CollectionImpl#isready)
[keys()](#keys-1)ts

```
keys(): IterableIterator<TKey>

```

```
keys(): IterableIterator<TKey>

```

Defined in:[packages/db/src/collection.ts:1002](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1002)

Get all keys (virtual derived state)
[Returns](#returns-19)

IterableIterator<TKey>
[Inherited from](#inherited-from-27)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[keys](/db/latest/docs/reference/classes/CollectionImpl#keys-1)
[map()](#map)ts

```
map<U>(callbackfn): U[]

```

```
map<U>(callbackfn): U[]

```

Defined in:[packages/db/src/collection.ts:1067](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1067)

Create a new array with the results of calling a function for each entry in the collection
[Type Parameters](#type-parameters-2)

•**U**
[Parameters](#parameters-9)[callbackfn](#callbackfn-1)

(value,key,index) =>U
[Returns](#returns-20)

U[]
[Inherited from](#inherited-from-28)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[map](/db/latest/docs/reference/classes/CollectionImpl#map)
[onFirstReady()](#onfirstready)ts

```
onFirstReady(callback): void

```

```
onFirstReady(callback): void

```

Defined in:[packages/db/src/collection.ts:272](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L272)

Register a callback to be executed when the collection first becomes ready
Useful for preloading collections
[Parameters](#parameters-10)[callback](#callback)

() =>void

Function to call when the collection first becomes ready
[Returns](#returns-21)

void
[Example](#example-4)ts

```
collection.onFirstReady(() => {
  console.log('Collection is ready for the first time')
  // Safe to access collection.state now
})

```

```
collection.onFirstReady(() => {
  console.log('Collection is ready for the first time')
  // Safe to access collection.state now
})

```[Inherited from](#inherited-from-29)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[onFirstReady](/db/latest/docs/reference/classes/CollectionImpl#onfirstready)
[onTransactionStateChange()](#ontransactionstatechange)ts

```
onTransactionStateChange(): void

```

```
onTransactionStateChange(): void

```

Defined in:[packages/db/src/collection.ts:2271](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2271)

Trigger a recomputation when transactions change
This method should be called by the Transaction class when state changes
[Returns](#returns-22)

void
[Inherited from](#inherited-from-30)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[onTransactionStateChange](/db/latest/docs/reference/classes/CollectionImpl#ontransactionstatechange)
[preload()](#preload)ts

```
preload(): Promise<void>

```

```
preload(): Promise<void>

```

Defined in:[packages/db/src/collection.ts:544](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L544)

Preload the collection data by starting sync if not already started
Multiple concurrent calls will share the same promise
[Returns](#returns-23)

Promise<void>
[Inherited from](#inherited-from-31)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[preload](/db/latest/docs/reference/classes/CollectionImpl#preload)
[startSyncImmediate()](#startsyncimmediate)ts

```
startSyncImmediate(): void

```

```
startSyncImmediate(): void

```

Defined in:[packages/db/src/collection.ts:450](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L450)

Start sync immediately - internal method for compiled queries
This bypasses lazy loading for special cases like live query results
[Returns](#returns-24)

void
[Inherited from](#inherited-from-32)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[startSyncImmediate](/db/latest/docs/reference/classes/CollectionImpl#startsyncimmediate)
[stateWhenReady()](#statewhenready)ts

```
stateWhenReady(): Promise<Map<TKey, T>>

```

```
stateWhenReady(): Promise<Map<TKey, T>>

```

Defined in:[packages/db/src/collection.ts:2052](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2052)

Gets the current state of the collection as a Map, but only resolves when data is available
Waits for the first sync commit to complete before resolving
[Returns](#returns-25)

Promise<Map<TKey,T>>

Promise that resolves to a Map containing all items in the collection
[Inherited from](#inherited-from-33)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[stateWhenReady](/db/latest/docs/reference/classes/CollectionImpl#statewhenready)
[subscribeChanges()](#subscribechanges)ts

```
subscribeChanges(callback, options): () => void

```

```
subscribeChanges(callback, options): () => void

```

Defined in:[packages/db/src/collection.ts:2158](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2158)

Subscribe to changes in the collection
[Parameters](#parameters-11)[callback](#callback-1)

(changes) =>void

Function called when items change
[options](#options-1)

[SubscribeChangesOptions](/db/latest/docs/reference/interfaces/subscribechangesoptions)<T> ={}

Subscription options including includeInitialState and where filter
[Returns](#returns-26)

Function

Unsubscribe function - Call this to stop listening for changes
[Returns](#returns-27)

void
[Examples](#examples-2)ts

```
// Basic subscription
const unsubscribe = collection.subscribeChanges((changes) => {
  changes.forEach(change => {
    console.log(`${change.type}: ${change.key}`, change.value)
  })
})

// Later: unsubscribe()

```

```
// Basic subscription
const unsubscribe = collection.subscribeChanges((changes) => {
  changes.forEach(change => {
    console.log(`${change.type}: ${change.key}`, change.value)
  })
})

// Later: unsubscribe()

```ts

```
// Include current state immediately
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, { includeInitialState: true })

```

```
// Include current state immediately
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, { includeInitialState: true })

```ts

```
// Subscribe only to changes matching a condition
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, {
  includeInitialState: true,
  where: (row) => row.status === 'active'
})

```

```
// Subscribe only to changes matching a condition
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, {
  includeInitialState: true,
  where: (row) => row.status === 'active'
})

```ts

```
// Subscribe using a pre-compiled expression
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, {
  includeInitialState: true,
  whereExpression: eq(row.status, 'active')
})

```

```
// Subscribe using a pre-compiled expression
const unsubscribe = collection.subscribeChanges((changes) => {
  updateUI(changes)
}, {
  includeInitialState: true,
  whereExpression: eq(row.status, 'active')
})

```[Inherited from](#inherited-from-34)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[subscribeChanges](/db/latest/docs/reference/classes/CollectionImpl#subscribechanges)
[subscribeChangesKey()](#subscribechangeskey)ts

```
subscribeChangesKey(
   key, 
   listener, 
   __namedParameters): () => void

```

```
subscribeChangesKey(
   key, 
   listener, 
   __namedParameters): () => void

```

Defined in:[packages/db/src/collection.ts:2197](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2197)

Subscribe to changes for a specific key
[Parameters](#parameters-12)[key](#key-3)

TKey
[listener](#listener)

[ChangeListener](/db/latest/docs/reference/type-aliases/changelistener)<T,TKey>
[__namedParameters](#__namedparameters)[includeInitialState?](#includeinitialstate)

boolean=false
[Returns](#returns-28)

Function
[Returns](#returns-29)

void
[Inherited from](#inherited-from-35)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[subscribeChangesKey](/db/latest/docs/reference/classes/CollectionImpl#subscribechangeskey)
[toArrayWhenReady()](#toarraywhenready)ts

```
toArrayWhenReady(): Promise<T[]>

```

```
toArrayWhenReady(): Promise<T[]>

```

Defined in:[packages/db/src/collection.ts:2081](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L2081)

Gets the current state of the collection as an Array, but only resolves when data is available
Waits for the first sync commit to complete before resolving
[Returns](#returns-30)

Promise<T[]>

Promise that resolves to an Array containing all items in the collection
[Inherited from](#inherited-from-36)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[toArrayWhenReady](/db/latest/docs/reference/classes/CollectionImpl#toarraywhenready)
[update()](#update)[Call Signature](#call-signature)ts

```
update<TItem>(key, callback): Transaction

```

```
update<TItem>(key, callback): Transaction

```

Defined in:[packages/db/src/collection.ts:1725](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1725)

Updates one or more items in the collection using a callback function
[Type Parameters](#type-parameters-3)

•**TItem***extends*object=T
[Parameters](#parameters-13)[key](#key-4)

unknown[]
[callback](#callback-2)

(drafts) =>void
[Returns](#returns-31)

[Transaction](/db/latest/docs/reference/classes/transaction)

A Transaction object representing the update operation(s)
[Throws](#throws-1)

If the updated data fails schema validation
[Examples](#examples-3)ts

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```ts

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```ts

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```[Inherited from](#inherited-from-37)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[update](/db/latest/docs/reference/classes/CollectionImpl#update)
[Call Signature](#call-signature-1)ts

```
update<TItem>(
   keys, 
   config, 
   callback): Transaction

```

```
update<TItem>(
   keys, 
   config, 
   callback): Transaction

```

Defined in:[packages/db/src/collection.ts:1731](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1731)

Updates one or more items in the collection using a callback function
[Type Parameters](#type-parameters-4)

•**TItem***extends*object=T
[Parameters](#parameters-14)[keys](#keys-2)

unknown[]

Single key or array of keys to update
[config](#config-4)

[OperationConfig](/db/latest/docs/reference/interfaces/operationconfig)
[callback](#callback-3)

(drafts) =>void
[Returns](#returns-32)

[Transaction](/db/latest/docs/reference/classes/transaction)

A Transaction object representing the update operation(s)
[Throws](#throws-2)

If the updated data fails schema validation
[Examples](#examples-4)ts

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```ts

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```ts

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```[Inherited from](#inherited-from-38)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[update](/db/latest/docs/reference/classes/CollectionImpl#update)
[Call Signature](#call-signature-2)ts

```
update<TItem>(id, callback): Transaction

```

```
update<TItem>(id, callback): Transaction

```

Defined in:[packages/db/src/collection.ts:1738](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1738)

Updates one or more items in the collection using a callback function
[Type Parameters](#type-parameters-5)

•**TItem***extends*object=T
[Parameters](#parameters-15)[id](#id-1)

unknown
[callback](#callback-4)

(draft) =>void
[Returns](#returns-33)

[Transaction](/db/latest/docs/reference/classes/transaction)

A Transaction object representing the update operation(s)
[Throws](#throws-3)

If the updated data fails schema validation
[Examples](#examples-5)ts

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```ts

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```ts

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```[Inherited from](#inherited-from-39)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[update](/db/latest/docs/reference/classes/CollectionImpl#update)
[Call Signature](#call-signature-3)ts

```
update<TItem>(
   id, 
   config, 
   callback): Transaction

```

```
update<TItem>(
   id, 
   config, 
   callback): Transaction

```

Defined in:[packages/db/src/collection.ts:1744](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1744)

Updates one or more items in the collection using a callback function
[Type Parameters](#type-parameters-6)

•**TItem***extends*object=T
[Parameters](#parameters-16)[id](#id-2)

unknown
[config](#config-5)

[OperationConfig](/db/latest/docs/reference/interfaces/operationconfig)
[callback](#callback-5)

(draft) =>void
[Returns](#returns-34)

[Transaction](/db/latest/docs/reference/classes/transaction)

A Transaction object representing the update operation(s)
[Throws](#throws-4)

If the updated data fails schema validation
[Examples](#examples-6)ts

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```

```
// Update single item by key
const tx = collection.update("todo-1", (draft) => {
  draft.completed = true
})
await tx.isPersisted.promise

```ts

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```

```
// Update multiple items
const tx = collection.update(["todo-1", "todo-2"], (drafts) => {
  drafts.forEach(draft => { draft.completed = true })
})
await tx.isPersisted.promise

```ts

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```

```
// Update with metadata
const tx = collection.update("todo-1",
  { metadata: { reason: "user update" } },
  (draft) => { draft.text = "Updated text" }
)
await tx.isPersisted.promise

```ts

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```

```
// Handle errors
try {
  const tx = collection.update("item-1", draft => { draft.value = "new" })
  await tx.isPersisted.promise
  console.log('Update successful')
} catch (error) {
  console.log('Update failed:', error)
}

```[Inherited from](#inherited-from-40)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[update](/db/latest/docs/reference/classes/CollectionImpl#update)
[values()](#values)ts

```
values(): IterableIterator<T>

```

```
values(): IterableIterator<T>

```

Defined in:[packages/db/src/collection.ts:1022](https://github.com/TanStack/db/blob/main/packages/db/src/collection.ts#L1022)

Get all values (virtual derived state)
[Returns](#returns-35)

IterableIterator<T>
[Inherited from](#inherited-from-41)

[CollectionImpl](/db/latest/docs/reference/classes/collectionimpl).[values](/db/latest/docs/reference/classes/CollectionImpl#values)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/interfaces/collection.md)[Core API Reference](/db/latest/docs/reference/index)[createCollection](/db/latest/docs/reference/functions/createcollection)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>