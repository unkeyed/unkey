# SortedMap | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: SortedMap<TKey, TValue>Type ParametersConstructorsnew SortedMap()Parameterscomparator?ReturnsAccessorssizeGet SignatureReturnsMethods[iterator]()Returnsclear()Returnsdelete()ParameterskeyReturnsentries()ReturnsforEach()ParameterscallbackfnReturnsget()ParameterskeyReturnshas()ParameterskeyReturnskeys()Returnsset()ParameterskeyvalueReturnsvalues()Returns# SortedMap

Copy Markdown[Class: SortedMap<TKey, TValue>](#class-sortedmaptkey-tvalue)

Defined in:[packages/db/src/SortedMap.ts:6](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L6)

A Map implementation that keeps its entries sorted based on a comparator function
[Type Parameters](#type-parameters)

•**TKey**

The type of keys in the map

•**TValue**

The type of values in the map
[Constructors](#constructors)[new SortedMap()](#new-sortedmap)ts

```
new SortedMap<TKey, TValue>(comparator?): SortedMap<TKey, TValue>

```

```
new SortedMap<TKey, TValue>(comparator?): SortedMap<TKey, TValue>

```

Defined in:[packages/db/src/SortedMap.ts:16](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L16)

Creates a new SortedMap instance
[Parameters](#parameters)[comparator?](#comparator)

(a,b) =>number

Optional function to compare values for sorting
[Returns](#returns)

[SortedMap](/db/latest/docs/reference/classes/sortedmap)<TKey,TValue>
[Accessors](#accessors)[size](#size)[Get Signature](#get-signature)ts

```
get size(): number

```

```
get size(): number

```

Defined in:[packages/db/src/SortedMap.ts:138](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L138)

Gets the number of key-value pairs in the map
[Returns](#returns-1)

number
[Methods](#methods)[[iterator]()](#iterator)ts

```
iterator: IterableIterator<[TKey, TValue]>

```

```
iterator: IterableIterator<[TKey, TValue]>

```

Defined in:[packages/db/src/SortedMap.ts:147](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L147)

Default iterator that returns entries in sorted order
[Returns](#returns-2)

IterableIterator<[TKey,TValue]>

An iterator for the map's entries
[clear()](#clear)ts

```
clear(): void

```

```
clear(): void

```

Defined in:[packages/db/src/SortedMap.ts:130](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L130)

Removes all key-value pairs from the map
[Returns](#returns-3)

void
[delete()](#delete)ts

```
delete(key): boolean

```

```
delete(key): boolean

```

Defined in:[packages/db/src/SortedMap.ts:106](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L106)

Removes a key-value pair from the map
[Parameters](#parameters-1)[key](#key)

TKey

The key to remove
[Returns](#returns-4)

boolean

True if the key was found and removed, false otherwise
[entries()](#entries)ts

```
entries(): IterableIterator<[TKey, TValue]>

```

```
entries(): IterableIterator<[TKey, TValue]>

```

Defined in:[packages/db/src/SortedMap.ts:158](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L158)

Returns an iterator for the map's entries in sorted order
[Returns](#returns-5)

IterableIterator<[TKey,TValue]>

An iterator for the map's entries
[forEach()](#foreach)ts

```
forEach(callbackfn): void

```

```
forEach(callbackfn): void

```

Defined in:[packages/db/src/SortedMap.ts:189](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L189)

Executes a callback function for each key-value pair in the map in sorted order
[Parameters](#parameters-2)[callbackfn](#callbackfn)

(value,key,map) =>void

Function to execute for each entry
[Returns](#returns-6)

void
[get()](#get)ts

```
get(key): undefined | TValue

```

```
get(key): undefined | TValue

```

Defined in:[packages/db/src/SortedMap.ts:96](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L96)

Gets a value by its key
[Parameters](#parameters-3)[key](#key-1)

TKey

The key to look up
[Returns](#returns-7)

undefined|TValue

The value associated with the key, or undefined if not found
[has()](#has)ts

```
has(key): boolean

```

```
has(key): boolean

```

Defined in:[packages/db/src/SortedMap.ts:123](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L123)

Checks if a key exists in the map
[Parameters](#parameters-4)[key](#key-2)

TKey

The key to check
[Returns](#returns-8)

boolean

True if the key exists, false otherwise
[keys()](#keys)ts

```
keys(): IterableIterator<TKey>

```

```
keys(): IterableIterator<TKey>

```

Defined in:[packages/db/src/SortedMap.ts:167](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L167)

Returns an iterator for the map's keys in sorted order
[Returns](#returns-9)

IterableIterator<TKey>

An iterator for the map's keys
[set()](#set)ts

```
set(key, value): this

```

```
set(key, value): this

```

Defined in:[packages/db/src/SortedMap.ts:73](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L73)

Sets a key-value pair in the map and maintains sort order
[Parameters](#parameters-5)[key](#key-3)

TKey

The key to set
[value](#value)

TValue

The value to associate with the key
[Returns](#returns-10)

this

This SortedMap instance for chaining
[values()](#values)ts

```
values(): IterableIterator<TValue>

```

```
values(): IterableIterator<TValue>

```

Defined in:[packages/db/src/SortedMap.ts:176](https://github.com/TanStack/db/blob/main/packages/db/src/SortedMap.ts#L176)

Returns an iterator for the map's values in sorted order
[Returns](#returns-11)

IterableIterator<TValue>

An iterator for the map's values[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/sortedmap.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>