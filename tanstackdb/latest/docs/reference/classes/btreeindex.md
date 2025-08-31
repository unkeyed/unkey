# BTreeIndex | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: BTreeIndex<TKey>ExtendsType ParametersConstructorsnew BTreeIndex()Parametersidexpressionname?options?ReturnsOverridesPropertiesexpressionInherited fromidInherited fromlastUpdatedInherited fromlookupCountInherited fromname?Inherited fromsupportedOperationsOverridestotalLookupTimeInherited fromAccessorsindexedKeysSetGet SignatureReturnskeyCountGet SignatureReturnsOverridesorderedEntriesArrayGet SignatureReturnsvalueMapDataGet SignatureReturnsMethodsadd()ParameterskeyitemReturnsOverridesbuild()ParametersentriesReturnsOverridesclear()ReturnsOverridesequalityLookup()ParametersvalueReturnsevaluateIndexExpression()ParametersitemReturnsInherited fromgetStats()ReturnsInherited frominArrayLookup()ParametersvaluesReturnsinitialize()Parameters_options?ReturnsOverrideslookup()ParametersoperationvalueReturnsOverridesmatchesField()ParametersfieldPathReturnsInherited fromrangeQuery()ParametersoptionsReturnsremove()ParameterskeyitemReturnsOverridessupports()ParametersoperationReturnsInherited fromtrackLookup()ParametersstartTimeReturnsInherited fromupdate()ParameterskeyoldItemnewItemReturnsOverridesupdateTimestamp()ReturnsInherited from# BTreeIndex

Copy Markdown[Class: BTreeIndex<TKey>](#class-btreeindextkey)

Defined in:[packages/db/src/indexes/btree-index.ts:28](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L28)

B+Tree index for sorted data with range queries
This maintains items in sorted order and provides efficient range operations
[Extends](#extends)

- [BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>
[Type Parameters](#type-parameters)

•**TKey***extends*string|number=string|number
[Constructors](#constructors)[new BTreeIndex()](#new-btreeindex)ts

```
new BTreeIndex<TKey>(
   id, 
   expression, 
   name?, 
options?): BTreeIndex<TKey>

```

```
new BTreeIndex<TKey>(
   id, 
   expression, 
   name?, 
options?): BTreeIndex<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:48](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L48)
[Parameters](#parameters)[id](#id)

number
[expression](#expression)

BasicExpression
[name?](#name)

string
[options?](#options)

any
[Returns](#returns)

[BTreeIndex](/db/latest/docs/reference/classes/btreeindex)<TKey>
[Overrides](#overrides)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[constructor](/db/latest/docs/reference/classes/BaseIndex#constructors)
[Properties](#properties)[expression](#expression-1)ts

```
readonly expression: BasicExpression;

```

```
readonly expression: BasicExpression;

```

Defined in:[packages/db/src/indexes/base-index.ts:33](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L33)
[Inherited from](#inherited-from)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[expression](/db/latest/docs/reference/classes/BaseIndex#expression-1)
[id](#id-1)ts

```
readonly id: number;

```

```
readonly id: number;

```

Defined in:[packages/db/src/indexes/base-index.ts:31](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L31)
[Inherited from](#inherited-from-1)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[id](/db/latest/docs/reference/classes/BaseIndex#id-1)
[lastUpdated](#lastupdated)ts

```
protected lastUpdated: Date;

```

```
protected lastUpdated: Date;

```

Defined in:[packages/db/src/indexes/base-index.ts:38](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L38)
[Inherited from](#inherited-from-2)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[lastUpdated](/db/latest/docs/reference/classes/BaseIndex#lastupdated)
[lookupCount](#lookupcount)ts

```
protected lookupCount: number = 0;

```

```
protected lookupCount: number = 0;

```

Defined in:[packages/db/src/indexes/base-index.ts:36](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L36)
[Inherited from](#inherited-from-3)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[lookupCount](/db/latest/docs/reference/classes/BaseIndex#lookupcount)
[name?](#name-1)ts

```
readonly optional name: string;

```

```
readonly optional name: string;

```

Defined in:[packages/db/src/indexes/base-index.ts:32](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L32)
[Inherited from](#inherited-from-4)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[name](/db/latest/docs/reference/classes/BaseIndex#name-1)
[supportedOperations](#supportedoperations)ts

```
readonly supportedOperations: Set<"eq" | "gt" | "gte" | "lt" | "lte" | "in" | "like" | "ilike">;

```

```
readonly supportedOperations: Set<"eq" | "gt" | "gte" | "lt" | "lte" | "in" | "like" | "ilike">;

```

Defined in:[packages/db/src/indexes/btree-index.ts:31](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L31)
[Overrides](#overrides-1)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[supportedOperations](/db/latest/docs/reference/classes/BaseIndex#supportedoperations)
[totalLookupTime](#totallookuptime)ts

```
protected totalLookupTime: number = 0;

```

```
protected totalLookupTime: number = 0;

```

Defined in:[packages/db/src/indexes/base-index.ts:37](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L37)
[Inherited from](#inherited-from-5)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[totalLookupTime](/db/latest/docs/reference/classes/BaseIndex#totallookuptime)
[Accessors](#accessors)[indexedKeysSet](#indexedkeysset)[Get Signature](#get-signature)ts

```
get indexedKeysSet(): Set<TKey>

```

```
get indexedKeysSet(): Set<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:250](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L250)
[Returns](#returns-1)

Set<TKey>
[keyCount](#keycount)[Get Signature](#get-signature-1)ts

```
get keyCount(): number

```

```
get keyCount(): number

```

Defined in:[packages/db/src/indexes/btree-index.ts:188](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L188)

Gets the number of indexed keys
[Returns](#returns-2)

number
[Overrides](#overrides-2)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[keyCount](/db/latest/docs/reference/classes/BaseIndex#keycount)
[orderedEntriesArray](#orderedentriesarray)[Get Signature](#get-signature-2)ts

```
get orderedEntriesArray(): [any, Set<TKey>][]

```

```
get orderedEntriesArray(): [any, Set<TKey>][]

```

Defined in:[packages/db/src/indexes/btree-index.ts:254](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L254)
[Returns](#returns-3)

[any,Set<TKey>][]
[valueMapData](#valuemapdata)[Get Signature](#get-signature-3)ts

```
get valueMapData(): Map<any, Set<TKey>>

```

```
get valueMapData(): Map<any, Set<TKey>>

```

Defined in:[packages/db/src/indexes/btree-index.ts:260](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L260)
[Returns](#returns-4)

Map<any,Set<TKey>>
[Methods](#methods)[add()](#add)ts

```
add(key, item): void

```

```
add(key, item): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:64](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L64)

Adds a value to the index
[Parameters](#parameters-1)[key](#key)

TKey
[item](#item)

any
[Returns](#returns-5)

void
[Overrides](#overrides-3)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[add](/db/latest/docs/reference/classes/BaseIndex#add)
[build()](#build)ts

```
build(entries): void

```

```
build(entries): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:132](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L132)

Builds the index from a collection of entries
[Parameters](#parameters-2)[entries](#entries)

Iterable<[TKey,any]>
[Returns](#returns-6)

void
[Overrides](#overrides-4)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[build](/db/latest/docs/reference/classes/BaseIndex#build)
[clear()](#clear)ts

```
clear(): void

```

```
clear(): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:143](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L143)

Clears all data from the index
[Returns](#returns-7)

void
[Overrides](#overrides-5)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[clear](/db/latest/docs/reference/classes/BaseIndex#clear)
[equalityLookup()](#equalitylookup)ts

```
equalityLookup(value): Set<TKey>

```

```
equalityLookup(value): Set<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:197](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L197)

Performs an equality lookup
[Parameters](#parameters-3)[value](#value)

any
[Returns](#returns-8)

Set<TKey>
[evaluateIndexExpression()](#evaluateindexexpression)ts

```
protected evaluateIndexExpression(item): any

```

```
protected evaluateIndexExpression(item): any

```

Defined in:[packages/db/src/indexes/base-index.ts:87](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L87)
[Parameters](#parameters-4)[item](#item-1)

any
[Returns](#returns-9)

any
[Inherited from](#inherited-from-6)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[evaluateIndexExpression](/db/latest/docs/reference/classes/BaseIndex#evaluateindexexpression)
[getStats()](#getstats)ts

```
getStats(): IndexStats

```

```
getStats(): IndexStats

```

Defined in:[packages/db/src/indexes/base-index.ts:74](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L74)
[Returns](#returns-10)

[IndexStats](/db/latest/docs/reference/interfaces/indexstats)
[Inherited from](#inherited-from-7)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[getStats](/db/latest/docs/reference/classes/BaseIndex#getstats)
[inArrayLookup()](#inarraylookup)ts

```
inArrayLookup(values): Set<TKey>

```

```
inArrayLookup(values): Set<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:236](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L236)

Performs an IN array lookup
[Parameters](#parameters-5)[values](#values)

any[]
[Returns](#returns-11)

Set<TKey>
[initialize()](#initialize)ts

```
protected initialize(_options?): void

```

```
protected initialize(_options?): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:59](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L59)
[Parameters](#parameters-6)[_options?](#_options)

[BTreeIndexOptions](/db/latest/docs/reference/interfaces/btreeindexoptions)
[Returns](#returns-12)

void
[Overrides](#overrides-6)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[initialize](/db/latest/docs/reference/classes/BaseIndex#initialize)
[lookup()](#lookup)ts

```
lookup(operation, value): Set<TKey>

```

```
lookup(operation, value): Set<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:153](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L153)

Performs a lookup operation
[Parameters](#parameters-7)[operation](#operation)

"eq"|"gt"|"gte"|"lt"|"lte"|"in"|"like"|"ilike"
[value](#value-1)

any
[Returns](#returns-13)

Set<TKey>
[Overrides](#overrides-7)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[lookup](/db/latest/docs/reference/classes/BaseIndex#lookup)
[matchesField()](#matchesfield)ts

```
matchesField(fieldPath): boolean

```

```
matchesField(fieldPath): boolean

```

Defined in:[packages/db/src/indexes/base-index.ts:66](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L66)
[Parameters](#parameters-8)[fieldPath](#fieldpath)

string[]
[Returns](#returns-14)

boolean
[Inherited from](#inherited-from-8)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[matchesField](/db/latest/docs/reference/classes/BaseIndex#matchesfield)
[rangeQuery()](#rangequery)ts

```
rangeQuery(options): Set<TKey>

```

```
rangeQuery(options): Set<TKey>

```

Defined in:[packages/db/src/indexes/btree-index.ts:205](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L205)

Performs a range query with options
This is more efficient for compound queries like "WHERE a > 5 AND a < 10"
[Parameters](#parameters-9)[options](#options-1)

[RangeQueryOptions](/db/latest/docs/reference/interfaces/rangequeryoptions)={}
[Returns](#returns-15)

Set<TKey>
[remove()](#remove)ts

```
remove(key, item): void

```

```
remove(key, item): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:92](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L92)

Removes a value from the index
[Parameters](#parameters-10)[key](#key-1)

TKey
[item](#item-2)

any
[Returns](#returns-16)

void
[Overrides](#overrides-8)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[remove](/db/latest/docs/reference/classes/BaseIndex#remove)
[supports()](#supports)ts

```
supports(operation): boolean

```

```
supports(operation): boolean

```

Defined in:[packages/db/src/indexes/base-index.ts:62](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L62)
[Parameters](#parameters-11)[operation](#operation-1)

"eq"|"gt"|"gte"|"lt"|"lte"|"in"|"like"|"ilike"
[Returns](#returns-17)

boolean
[Inherited from](#inherited-from-9)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[supports](/db/latest/docs/reference/classes/BaseIndex#supports)
[trackLookup()](#tracklookup)ts

```
protected trackLookup(startTime): void

```

```
protected trackLookup(startTime): void

```

Defined in:[packages/db/src/indexes/base-index.ts:92](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L92)
[Parameters](#parameters-12)[startTime](#starttime)

number
[Returns](#returns-18)

void
[Inherited from](#inherited-from-10)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[trackLookup](/db/latest/docs/reference/classes/BaseIndex#tracklookup)
[update()](#update)ts

```
update(
   key, 
   oldItem, 
   newItem): void

```

```
update(
   key, 
   oldItem, 
   newItem): void

```

Defined in:[packages/db/src/indexes/btree-index.ts:124](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/btree-index.ts#L124)

Updates a value in the index
[Parameters](#parameters-13)[key](#key-2)

TKey
[oldItem](#olditem)

any
[newItem](#newitem)

any
[Returns](#returns-19)

void
[Overrides](#overrides-9)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[update](/db/latest/docs/reference/classes/BaseIndex#update)
[updateTimestamp()](#updatetimestamp)ts

```
protected updateTimestamp(): void

```

```
protected updateTimestamp(): void

```

Defined in:[packages/db/src/indexes/base-index.ts:98](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L98)
[Returns](#returns-20)

void
[Inherited from](#inherited-from-11)

[BaseIndex](/db/latest/docs/reference/classes/baseindex).[updateTimestamp](/db/latest/docs/reference/classes/BaseIndex#updatetimestamp)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/btreeindex.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>