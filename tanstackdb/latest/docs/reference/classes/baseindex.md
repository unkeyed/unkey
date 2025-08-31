# BaseIndex | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: abstract BaseIndex<TKey>Extended byType ParametersConstructorsnew BaseIndex()Parametersidexpressionname?options?ReturnsPropertiesexpressionidlastUpdatedlookupCountname?supportedOperationstotalLookupTimeAccessorskeyCountGet SignatureReturnsMethodsadd()ParameterskeyitemReturnsbuild()ParametersentriesReturnsclear()ReturnsevaluateIndexExpression()ParametersitemReturnsgetStats()Returnsinitialize()Parametersoptions?Returnslookup()ParametersoperationvalueReturnsmatchesField()ParametersfieldPathReturnsremove()ParameterskeyitemReturnssupports()ParametersoperationReturnstrackLookup()ParametersstartTimeReturnsupdate()ParameterskeyoldItemnewItemReturnsupdateTimestamp()Returns# BaseIndex

Copy Markdown[Class: abstract BaseIndex<TKey>](#class-abstract-baseindextkey)

Defined in:[packages/db/src/indexes/base-index.ts:28](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L28)

Base abstract class that all index types extend
[Extended by](#extended-by)

- [BTreeIndex](/db/latest/docs/reference/classes/btreeindex)
[Type Parameters](#type-parameters)

•**TKey***extends*string|number=string|number
[Constructors](#constructors)[new BaseIndex()](#new-baseindex)ts

```
new BaseIndex<TKey>(
   id, 
   expression, 
   name?, 
options?): BaseIndex<TKey>

```

```
new BaseIndex<TKey>(
   id, 
   expression, 
   name?, 
options?): BaseIndex<TKey>

```

Defined in:[packages/db/src/indexes/base-index.ts:40](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L40)
[Parameters](#parameters)[id](#id)

number
[expression](#expression)

BasicExpression
[name?](#name)

string
[options?](#options)

any
[Returns](#returns)

[BaseIndex](/db/latest/docs/reference/classes/baseindex)<TKey>
[Properties](#properties)[expression](#expression-1)ts

```
readonly expression: BasicExpression;

```

```
readonly expression: BasicExpression;

```

Defined in:[packages/db/src/indexes/base-index.ts:33](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L33)
[id](#id-1)ts

```
readonly id: number;

```

```
readonly id: number;

```

Defined in:[packages/db/src/indexes/base-index.ts:31](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L31)
[lastUpdated](#lastupdated)ts

```
protected lastUpdated: Date;

```

```
protected lastUpdated: Date;

```

Defined in:[packages/db/src/indexes/base-index.ts:38](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L38)
[lookupCount](#lookupcount)ts

```
protected lookupCount: number = 0;

```

```
protected lookupCount: number = 0;

```

Defined in:[packages/db/src/indexes/base-index.ts:36](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L36)
[name?](#name-1)ts

```
readonly optional name: string;

```

```
readonly optional name: string;

```

Defined in:[packages/db/src/indexes/base-index.ts:32](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L32)
[supportedOperations](#supportedoperations)ts

```
abstract readonly supportedOperations: Set<"eq" | "gt" | "gte" | "lt" | "lte" | "in" | "like" | "ilike">;

```

```
abstract readonly supportedOperations: Set<"eq" | "gt" | "gte" | "lt" | "lte" | "in" | "like" | "ilike">;

```

Defined in:[packages/db/src/indexes/base-index.ts:34](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L34)
[totalLookupTime](#totallookuptime)ts

```
protected totalLookupTime: number = 0;

```

```
protected totalLookupTime: number = 0;

```

Defined in:[packages/db/src/indexes/base-index.ts:37](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L37)
[Accessors](#accessors)[keyCount](#keycount)[Get Signature](#get-signature)ts

```
get abstract keyCount(): number

```

```
get abstract keyCount(): number

```

Defined in:[packages/db/src/indexes/base-index.ts:59](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L59)
[Returns](#returns-1)

number
[Methods](#methods)[add()](#add)ts

```
abstract add(key, item): void

```

```
abstract add(key, item): void

```

Defined in:[packages/db/src/indexes/base-index.ts:53](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L53)
[Parameters](#parameters-1)[key](#key)

TKey
[item](#item)

any
[Returns](#returns-2)

void
[build()](#build)ts

```
abstract build(entries): void

```

```
abstract build(entries): void

```

Defined in:[packages/db/src/indexes/base-index.ts:56](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L56)
[Parameters](#parameters-2)[entries](#entries)

Iterable<[TKey,any]>
[Returns](#returns-3)

void
[clear()](#clear)ts

```
abstract clear(): void

```

```
abstract clear(): void

```

Defined in:[packages/db/src/indexes/base-index.ts:57](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L57)
[Returns](#returns-4)

void
[evaluateIndexExpression()](#evaluateindexexpression)ts

```
protected evaluateIndexExpression(item): any

```

```
protected evaluateIndexExpression(item): any

```

Defined in:[packages/db/src/indexes/base-index.ts:87](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L87)
[Parameters](#parameters-3)[item](#item-1)

any
[Returns](#returns-5)

any
[getStats()](#getstats)ts

```
getStats(): IndexStats

```

```
getStats(): IndexStats

```

Defined in:[packages/db/src/indexes/base-index.ts:74](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L74)
[Returns](#returns-6)

[IndexStats](/db/latest/docs/reference/interfaces/indexstats)
[initialize()](#initialize)ts

```
abstract protected initialize(options?): void

```

```
abstract protected initialize(options?): void

```

Defined in:[packages/db/src/indexes/base-index.ts:85](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L85)
[Parameters](#parameters-4)[options?](#options-1)

any
[Returns](#returns-7)

void
[lookup()](#lookup)ts

```
abstract lookup(operation, value): Set<TKey>

```

```
abstract lookup(operation, value): Set<TKey>

```

Defined in:[packages/db/src/indexes/base-index.ts:58](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L58)
[Parameters](#parameters-5)[operation](#operation)

"eq"|"gt"|"gte"|"lt"|"lte"|"in"|"like"|"ilike"
[value](#value)

any
[Returns](#returns-8)

Set<TKey>
[matchesField()](#matchesfield)ts

```
matchesField(fieldPath): boolean

```

```
matchesField(fieldPath): boolean

```

Defined in:[packages/db/src/indexes/base-index.ts:66](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L66)
[Parameters](#parameters-6)[fieldPath](#fieldpath)

string[]
[Returns](#returns-9)

boolean
[remove()](#remove)ts

```
abstract remove(key, item): void

```

```
abstract remove(key, item): void

```

Defined in:[packages/db/src/indexes/base-index.ts:54](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L54)
[Parameters](#parameters-7)[key](#key-1)

TKey
[item](#item-2)

any
[Returns](#returns-10)

void
[supports()](#supports)ts

```
supports(operation): boolean

```

```
supports(operation): boolean

```

Defined in:[packages/db/src/indexes/base-index.ts:62](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L62)
[Parameters](#parameters-8)[operation](#operation-1)

"eq"|"gt"|"gte"|"lt"|"lte"|"in"|"like"|"ilike"
[Returns](#returns-11)

boolean
[trackLookup()](#tracklookup)ts

```
protected trackLookup(startTime): void

```

```
protected trackLookup(startTime): void

```

Defined in:[packages/db/src/indexes/base-index.ts:92](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L92)
[Parameters](#parameters-9)[startTime](#starttime)

number
[Returns](#returns-12)

void
[update()](#update)ts

```
abstract update(
   key, 
   oldItem, 
   newItem): void

```

```
abstract update(
   key, 
   oldItem, 
   newItem): void

```

Defined in:[packages/db/src/indexes/base-index.ts:55](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L55)
[Parameters](#parameters-10)[key](#key-2)

TKey
[oldItem](#olditem)

any
[newItem](#newitem)

any
[Returns](#returns-13)

void
[updateTimestamp()](#updatetimestamp)ts

```
protected updateTimestamp(): void

```

```
protected updateTimestamp(): void

```

Defined in:[packages/db/src/indexes/base-index.ts:98](https://github.com/TanStack/db/blob/main/packages/db/src/indexes/base-index.ts#L98)
[Returns](#returns-14)

void[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/baseindex.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>