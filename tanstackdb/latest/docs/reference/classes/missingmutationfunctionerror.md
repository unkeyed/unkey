# MissingMutationFunctionError | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: MissingMutationFunctionErrorExtendsConstructorsnew MissingMutationFunctionError()ReturnsOverridesPropertiescause?Inherited frommessageInherited fromnameInherited fromstack?Inherited fromprepareStackTrace()?ParameterserrstackTracesReturnsSeeInherited fromstackTraceLimitInherited fromMethodscaptureStackTrace()ParameterstargetObjectconstructorOpt?ReturnsInherited from# MissingMutationFunctionError

Copy Markdown[Class: MissingMutationFunctionError](#class-missingmutationfunctionerror)

Defined in:[packages/db/src/errors.ts:226](https://github.com/TanStack/db/blob/main/packages/db/src/errors.ts#L226)
[Extends](#extends)

- [TransactionError](/db/latest/docs/reference/classes/transactionerror)
[Constructors](#constructors)[new MissingMutationFunctionError()](#new-missingmutationfunctionerror)ts

```
new MissingMutationFunctionError(): MissingMutationFunctionError

```

```
new MissingMutationFunctionError(): MissingMutationFunctionError

```

Defined in:[packages/db/src/errors.ts:227](https://github.com/TanStack/db/blob/main/packages/db/src/errors.ts#L227)
[Returns](#returns)

[MissingMutationFunctionError](/db/latest/docs/reference/classes/missingmutationfunctionerror)
[Overrides](#overrides)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[constructor](/db/latest/docs/reference/classes/TransactionError#constructors)
[Properties](#properties)[cause?](#cause)ts

```
optional cause: unknown;

```

```
optional cause: unknown;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es2022.error.d.ts:26
[Inherited from](#inherited-from)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[cause](/db/latest/docs/reference/classes/TransactionError#cause)
[message](#message)ts

```
message: string;

```

```
message: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1077
[Inherited from](#inherited-from-1)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[message](/db/latest/docs/reference/classes/TransactionError#message-1)
[name](#name)ts

```
name: string;

```

```
name: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1076
[Inherited from](#inherited-from-2)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[name](/db/latest/docs/reference/classes/TransactionError#name)
[stack?](#stack)ts

```
optional stack: string;

```

```
optional stack: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1078
[Inherited from](#inherited-from-3)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[stack](/db/latest/docs/reference/classes/TransactionError#stack)
[prepareStackTrace()?](#preparestacktrace)ts

```
static optional prepareStackTrace: (err, stackTraces) => any;

```

```
static optional prepareStackTrace: (err, stackTraces) => any;

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:143

Optional override for formatting stack traces
[Parameters](#parameters)[err](#err)

Error
[stackTraces](#stacktraces)

CallSite[]
[Returns](#returns-1)

any
[See](#see)

[https://v8.dev/docs/stack-trace-api#customizing-stack-traces](https://v8.dev/docs/stack-trace-api#customizing-stack-traces)
[Inherited from](#inherited-from-4)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[prepareStackTrace](/db/latest/docs/reference/classes/TransactionError#preparestacktrace)
[stackTraceLimit](#stacktracelimit)ts

```
static stackTraceLimit: number;

```

```
static stackTraceLimit: number;

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:145
[Inherited from](#inherited-from-5)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[stackTraceLimit](/db/latest/docs/reference/classes/TransactionError#stacktracelimit)
[Methods](#methods)[captureStackTrace()](#capturestacktrace)ts

```
static captureStackTrace(targetObject, constructorOpt?): void

```

```
static captureStackTrace(targetObject, constructorOpt?): void

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:136

Create .stack property on a target object
[Parameters](#parameters-1)[targetObject](#targetobject)

object
[constructorOpt?](#constructoropt)

Function
[Returns](#returns-2)

void
[Inherited from](#inherited-from-6)

[TransactionError](/db/latest/docs/reference/classes/transactionerror).[captureStackTrace](/db/latest/docs/reference/classes/TransactionError#capturestacktrace)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/missingmutationfunctionerror.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>