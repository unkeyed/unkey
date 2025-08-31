# QueryOptimizerError | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageClass: QueryOptimizerErrorExtendsExtended byConstructorsnew QueryOptimizerError()ParametersmessageReturnsOverridesPropertiescause?Inherited frommessageInherited fromnameInherited fromstack?Inherited fromprepareStackTrace()?ParameterserrstackTracesReturnsSeeInherited fromstackTraceLimitInherited fromMethodscaptureStackTrace()ParameterstargetObjectconstructorOpt?ReturnsInherited from# QueryOptimizerError

Copy Markdown[Class: QueryOptimizerError](#class-queryoptimizererror)

Defined in:[packages/db/src/errors.ts:539](https://github.com/TanStack/db/blob/main/packages/db/src/errors.ts#L539)
[Extends](#extends)

- [TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror)
[Extended by](#extended-by)

- [CannotCombineEmptyExpressionListError](/db/latest/docs/reference/classes/cannotcombineemptyexpressionlisterror)
[Constructors](#constructors)[new QueryOptimizerError()](#new-queryoptimizererror)ts

```
new QueryOptimizerError(message): QueryOptimizerError

```

```
new QueryOptimizerError(message): QueryOptimizerError

```

Defined in:[packages/db/src/errors.ts:540](https://github.com/TanStack/db/blob/main/packages/db/src/errors.ts#L540)
[Parameters](#parameters)[message](#message)

string
[Returns](#returns)

[QueryOptimizerError](/db/latest/docs/reference/classes/queryoptimizererror)
[Overrides](#overrides)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[constructor](/db/latest/docs/reference/classes/TanStackDBError#constructors)
[Properties](#properties)[cause?](#cause)ts

```
optional cause: unknown;

```

```
optional cause: unknown;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es2022.error.d.ts:26
[Inherited from](#inherited-from)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[cause](/db/latest/docs/reference/classes/TanStackDBError#cause)
[message](#message-1)ts

```
message: string;

```

```
message: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1077
[Inherited from](#inherited-from-1)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[message](/db/latest/docs/reference/classes/TanStackDBError#message-1)
[name](#name)ts

```
name: string;

```

```
name: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1076
[Inherited from](#inherited-from-2)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[name](/db/latest/docs/reference/classes/TanStackDBError#name)
[stack?](#stack)ts

```
optional stack: string;

```

```
optional stack: string;

```

Defined in: node_modules/.pnpm/[typescript@5.8.2](mailto:typescript@5.8.2)/node_modules/typescript/lib/lib.es5.d.ts:1078
[Inherited from](#inherited-from-3)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[stack](/db/latest/docs/reference/classes/TanStackDBError#stack)
[prepareStackTrace()?](#preparestacktrace)ts

```
static optional prepareStackTrace: (err, stackTraces) => any;

```

```
static optional prepareStackTrace: (err, stackTraces) => any;

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:143

Optional override for formatting stack traces
[Parameters](#parameters-1)[err](#err)

Error
[stackTraces](#stacktraces)

CallSite[]
[Returns](#returns-1)

any
[See](#see)

[https://v8.dev/docs/stack-trace-api#customizing-stack-traces](https://v8.dev/docs/stack-trace-api#customizing-stack-traces)
[Inherited from](#inherited-from-4)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[prepareStackTrace](/db/latest/docs/reference/classes/TanStackDBError#preparestacktrace)
[stackTraceLimit](#stacktracelimit)ts

```
static stackTraceLimit: number;

```

```
static stackTraceLimit: number;

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:145
[Inherited from](#inherited-from-5)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[stackTraceLimit](/db/latest/docs/reference/classes/TanStackDBError#stacktracelimit)
[Methods](#methods)[captureStackTrace()](#capturestacktrace)ts

```
static captureStackTrace(targetObject, constructorOpt?): void

```

```
static captureStackTrace(targetObject, constructorOpt?): void

```

Defined in: node_modules/.pnpm/@[types+node@22.13.10](mailto:types+node@22.13.10)/node_modules/@types/node/globals.d.ts:136

Create .stack property on a target object
[Parameters](#parameters-2)[targetObject](#targetobject)

object
[constructorOpt?](#constructoropt)

Function
[Returns](#returns-2)

void
[Inherited from](#inherited-from-6)

[TanStackDBError](/db/latest/docs/reference/classes/tanstackdberror).[captureStackTrace](/db/latest/docs/reference/classes/TanStackDBError#capturestacktrace)[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/reference/classes/queryoptimizererror.md)[Home](/db/latest)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>