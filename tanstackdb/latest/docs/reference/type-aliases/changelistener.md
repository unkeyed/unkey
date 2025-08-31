# ChangeListener | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageType Alias: ChangeListener()<T, TKey>Type ParametersParameterschangesReturnsExamples# ChangeListener

Copy Markdown[Type Alias: ChangeListener()<T, TKey>](#type-alias-changelistenert-tkey)ts

```
type ChangeListener<T, TKey> = (changes) => void;

```

```
type ChangeListener<T, TKey> = (changes) => void;

```

Defined in:[packages/db/src/types.ts:627](https://github.com/TanStack/db/blob/main/packages/db/src/types.ts#L627)

Function type for listening to collection changes
[Type Parameters](#type-parameters)

•**T***extends*object=Record<string,unknown>

•**TKey***extends*string|number=string|number
[Parameters](#parameters)[changes](#changes)

[ChangeMessage](/db/latest/docs/reference/interfaces/changemessage)<T,TKey>[]

Array of change messages describing what happened
[Returns](#returns)

void
[Examples](#examples)ts

```
// Basic change listener
const listener: ChangeListener = (changes) => {
  changes.forEach(change => {
    console.log(`${change.type}: ${change.key}`, change.value)
  })
}

collection.subscribeChanges(listener)

```

```
// Basic change listener
const listener: ChangeListener = (changes) => {
  changes.forEach(change => {
    console.log(`${change.type}: ${change.key}`, change.value)
  })
}

collection.subscribeChanges(listener)

```ts

```
// Handle different change types
const listener: ChangeListener<Todo> = (changes) => {
  for (const change of changes) {
    switch (change.type) {
      case 'insert':
        addToUI(change.value)
        break
      case 'update':
        updateInUI(change.key, change.value, change.previousValue)
        break
      case 'delete':
        removeFromUI(change.key)
        break
    }
  }
}

```

```
// Handle different change types
const listener: ChangeListener<Todo> = (changes) => {
  for (const change of changes) {
    switch (change.type) {
      case 'insert':
        addToUI(change.value)
        break
      case 'update':
        updateInUI(change.key, change.value, change.previousValue)
        break
      case 'delete':
        removeFromUI(change.key)
        break
    }
  }
}

```[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/type-aliases/changelistener.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>