# Live Queries | TanStack DB Docs

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
- [useLiveQueryreact](/db/latest/docs/framework/react/reference/functions/uselivequery)On this pageTanStack DB Live QueriesTable of ContentsCreating Live Query CollectionsUsing liveQueryCollectionOptionsConfiguration OptionsConvenience FunctionUsing with FrameworksFrom ClauseMethod SignatureBasic UsageWhere ClausesMethod SignatureBasic FilteringMultiple ConditionsComplex ConditionsAvailable OperatorsSelectMethod SignatureBasic SelectsField RenamingComputed FieldsUsing Functions and Including All FieldsJoinsMethod SignatureBasic JoinsJoin TypesLeft JoinRight JoinInner JoinFull JoinMultiple JoinsSubqueriesSubqueries in from ClausesSubqueries in join ClausesSubquery deduplicationComplex Nested SubqueriesgroupBy and AggregationsMethod SignatureBasic GroupingMultiple Column GroupingAggregate FunctionsHaving ClausesMethod SignatureImplicit Single-Group AggregationAccessing Grouped DataOrder By, Limit, and OffsetMethod SignaturesBasic OrderingMultiple Column OrderingDescending OrderPagination with limit and offsetComposable QueriesConditional Query BuildingCaching Intermediate ResultsReusable Query DefinitionsReusable Callback FunctionsExpression Functions ReferenceComparison Operatorseq(left, right)gt(left, right), gte(left, right), lt(left, right), lte(left, right)inArray(value, array)like(value, pattern), ilike(value, pattern)Logical Operatorsand(...conditions)or(...conditions)not(condition)String Functionsupper(value), lower(value)length(value)concat(...values)Mathematical Functionsadd(left, right)coalesce(...values)Aggregate Functionscount(value)sum(value)avg(value)min(value), max(value)Function CompositionFunctional VariantsFunctional SelectFunctional WhereFunctional HavingComplex TransformationsType InferenceWhen to Use Functional Variants# Live Queries

Copy Markdown[TanStack DB Live Queries](#tanstack-db-live-queries)

TanStack DB provides a powerful, type-safe query system that allows you to fetch, filter, transform, and aggregate data from collections using a SQL-like fluent API. All queries are**live**by default, meaning they automatically update when the underlying data changes.

The query system is built around an API similar to SQL query builders like Kysely or Drizzle where you chain methods together to compose your query. The query builder doesn't perform operations in the order of method calls - instead, it composes your query into an optimal incremental pipeline that gets compiled and executed efficiently. Each method returns a new query builder, allowing you to chain operations together.

Live queries resolve to collections that automatically update when their underlying data changes. You can subscribe to changes, iterate over results, and use all the standard collection methods.
ts

```
import { createCollection, liveQueryCollectionOptions, eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
        email: user.email,
      }))
}))

```

```
import { createCollection, liveQueryCollectionOptions, eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
        email: user.email,
      }))
}))

```

The result types are automatically inferred from your query structure, providing full TypeScript support. When you use aselectclause, the result type matches your projection. Withoutselect, you get the full schema with proper join optionality.
[Table of Contents](#table-of-contents)

- [Creating Live Query Collections](#creating-live-query-collections)
- [From Clause](#from-clause)
- [Where Clauses](#where-clauses)
- [Select Projections](#select)
- [Joins](#joins)
- [Subqueries](#subqueries)
- [groupBy and Aggregations](#groupby-and-aggregations)
- [Order By, Limit, and Offset](#order-by-limit-and-offset)
- [Composable Queries](#composable-queries)
- [Expression Functions Reference](#expression-functions-reference)
- [Functional Variants](#functional-variants)
[Creating Live Query Collections](#creating-live-query-collections)

To create a live query collection, you can useliveQueryCollectionOptionswithcreateCollection, or use the convenience functioncreateLiveQueryCollection.
[Using liveQueryCollectionOptions](#using-livequerycollectionoptions)

The fundamental way to create a live query is usingliveQueryCollectionOptionswithcreateCollection:
ts

```
import { createCollection, liveQueryCollectionOptions, eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
      }))
}))

```

```
import { createCollection, liveQueryCollectionOptions, eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
      }))
}))

```[Configuration Options](#configuration-options)

For more control, you can specify additional options:
ts

```
const activeUsers = createCollection(liveQueryCollectionOptions({
  id: 'active-users', // Optional: auto-generated if not provided
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
      })),
  getKey: (user) => user.id, // Optional: uses stream key if not provided
  startSync: true, // Optional: starts sync immediately
}))

```

```
const activeUsers = createCollection(liveQueryCollectionOptions({
  id: 'active-users', // Optional: auto-generated if not provided
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
      })),
  getKey: (user) => user.id, // Optional: uses stream key if not provided
  startSync: true, // Optional: starts sync immediately
}))

```OptionTypeDescriptionidstring (optional)An optional unique identifier for the live query. If not provided, it will be auto-generated. This is useful for debugging and logging.queryQueryBuilder or functionThe query definition, this is either a Query instance or a function that returns a Query instance.getKey(item) => string | number (optional)A function that extracts a unique key from each row. If not provided, the stream's internal key will be used. For simple cases this is the key from the parent collection, but in the case of joins, the auto-generated key will be a composite of the parent keys. Using getKey is useful when you want to use a specific key from a parent collection for the resulting collection.schemaSchema (optional)Optional schema for validationstartSyncboolean (optional)Whether to start syncing immediately. Defaults to true.gcTimenumber (optional)Garbage collection time in milliseconds. Defaults to 5000 (5 seconds).[Convenience Function](#convenience-function)

For simpler cases, you can usecreateLiveQueryCollectionas a shortcut:
ts

```
import { createLiveQueryCollection, eq } from '@tanstack/db'

const activeUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```

```
import { createLiveQueryCollection, eq } from '@tanstack/db'

const activeUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```[Using with Frameworks](#using-with-frameworks)

In React, you can use theuseLiveQueryhook:
tsx

```
import { useLiveQuery } from '@tanstack/react-db'

function UserList() {
  const activeUsers = useLiveQuery((q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
  )

  return (
    <ul>
      {activeUsers.map(user => (
        <li key={user.id}>{user.name}</li>
      ))}
    </ul>
  )
}

```

```
import { useLiveQuery } from '@tanstack/react-db'

function UserList() {
  const activeUsers = useLiveQuery((q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
  )

  return (
    <ul>
      {activeUsers.map(user => (
        <li key={user.id}>{user.name}</li>
      ))}
    </ul>
  )
}

```

For more details on framework integration, see the[React](/db/latest/docs/framework/react/adapter)and[Vue](/db/latest/docs/framework/vue/adapter)adapter documentation.
[From Clause](#from-clause)

The foundation of every query is thefrommethod, which specifies the source collection or subquery. You can alias the source using object syntax.
[Method Signature](#method-signature)ts

```
from({
  [alias]: Collection | Query,
}): Query

```

```
from({
  [alias]: Collection | Query,
}): Query

```

**Parameters:**

- [alias]- A Collection or Query instance. Note that only a single aliased collection or subquery is allowed in thefromclause.
[Basic Usage](#basic-usage)

Start with a basic query that selects all records from a collection:
ts

```
const allUsers = createCollection(liveQueryCollectionOptions({
  query: (q) => q.from({ user: usersCollection })
}))

```

```
const allUsers = createCollection(liveQueryCollectionOptions({
  query: (q) => q.from({ user: usersCollection })
}))

```

The result contains all users with their full schema. You can iterate over the results or access them by key:
ts

```
// Get all users as an array
const users = allUsers.toArray

// Get a specific user by ID
const user = allUsers.get(1)

// Check if a user exists
const hasUser = allUsers.has(1)

```

```
// Get all users as an array
const users = allUsers.toArray

// Get a specific user by ID
const user = allUsers.get(1)

// Check if a user exists
const hasUser = allUsers.has(1)

```

Use aliases to make your queries more readable, especially when working with multiple collections:
ts

```
const users = createCollection(liveQueryCollectionOptions({
  query: (q) => q.from({ u: usersCollection })
}))

// Access fields using the alias
const userNames = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ u: usersCollection })
      .select(({ u }) => ({
        name: u.name,
        email: u.email,
      }))
}))

```

```
const users = createCollection(liveQueryCollectionOptions({
  query: (q) => q.from({ u: usersCollection })
}))

// Access fields using the alias
const userNames = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ u: usersCollection })
      .select(({ u }) => ({
        name: u.name,
        email: u.email,
      }))
}))

```[Where Clauses](#where-clauses)

Usewhereclauses to filter your data based on conditions. You can chain multiplewherecalls - they are combined withandlogic.

Thewheremethod takes a callback function that receives an object containing your table aliases and returns a boolean expression. You build these expressions using comparison functions likeeq(),gt(), and logical operators likeand()andor(). This declarative approach allows the query system to optimize your filters efficiently. These are described in more detail in the[Expression Functions Reference](#expression-functions-reference)section. This is very similar to how you construct queries using Kysely or Drizzle.

It's important to note that thewheremethod is not a function that is executed on each row or the results, its a way to describe the query that will be executed. This declarative approach works well for almost all use cases, but if you need to use a more complex condition, there is the functional variant asfn.wherewhich is described in the[Functional Variants](#functional-variants)section.
[Method Signature](#method-signature-1)ts

```
where(
  condition: (row: TRow) => Expression<boolean>
): Query

```

```
where(
  condition: (row: TRow) => Expression<boolean>
): Query

```

**Parameters:**

- condition- A callback function that receives the row object with table aliases and returns a boolean expression
[Basic Filtering](#basic-filtering)

Filter users by a simple condition:
ts

```
import { eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
}))

```

```
import { eq } from '@tanstack/db'

const activeUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
}))

```[Multiple Conditions](#multiple-conditions)

Chain multiplewherecalls for AND logic:
ts

```
import { eq, gt } from '@tanstack/db'

const adultActiveUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .where(({ user }) => gt(user.age, 18))
}))

```

```
import { eq, gt } from '@tanstack/db'

const adultActiveUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
      .where(({ user }) => gt(user.age, 18))
}))

```[Complex Conditions](#complex-conditions)

Use logical operators to build complex conditions:
ts

```
import { eq, gt, or, and } from '@tanstack/db'

const specialUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => 
      and(
        eq(user.active, true),
        or(
          gt(user.age, 25),
          eq(user.role, 'admin')
        )
      )
    )
)

```

```
import { eq, gt, or, and } from '@tanstack/db'

const specialUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => 
      and(
        eq(user.active, true),
        or(
          gt(user.age, 25),
          eq(user.role, 'admin')
        )
      )
    )
)

```[Available Operators](#available-operators)

The query system provides several comparison operators:
ts

```
import { eq, gt, gte, lt, lte, like, ilike, inArray, and, or, not } from '@tanstack/db'

// Equality
eq(user.id, 1)

// Comparisons
gt(user.age, 18)    // greater than
gte(user.age, 18)   // greater than or equal
lt(user.age, 65)    // less than
lte(user.age, 65)   // less than or equal

// String matching
like(user.name, 'John%')    // case-sensitive pattern matching
ilike(user.name, 'john%')   // case-insensitive pattern matching

// Array membership
inArray(user.id, [1, 2, 3])

// Logical operators
and(condition1, condition2)
or(condition1, condition2)
not(condition)

```

```
import { eq, gt, gte, lt, lte, like, ilike, inArray, and, or, not } from '@tanstack/db'

// Equality
eq(user.id, 1)

// Comparisons
gt(user.age, 18)    // greater than
gte(user.age, 18)   // greater than or equal
lt(user.age, 65)    // less than
lte(user.age, 65)   // less than or equal

// String matching
like(user.name, 'John%')    // case-sensitive pattern matching
ilike(user.name, 'john%')   // case-insensitive pattern matching

// Array membership
inArray(user.id, [1, 2, 3])

// Logical operators
and(condition1, condition2)
or(condition1, condition2)
not(condition)

```

For a complete reference of all available functions, see the[Expression Functions Reference](#expression-functions-reference)section.
[Select](#select)

Useselectto specify which fields to include in your results and transform your data. Withoutselect, you get the full schema.

Similar to thewhereclause, theselectmethod takes a callback function that receives an object containing your table aliases and returns an object with the fields you want to include in your results. These can be combined with functions from the[Expression Functions Reference](#expression-functions-reference)section to create computed fields. You can also use the spread operator to include all fields from a table.
[Method Signature](#method-signature-2)ts

```
select(
  projection: (row: TRow) => Record<string, Expression>
): Query

```

```
select(
  projection: (row: TRow) => Record<string, Expression>
): Query

```

**Parameters:**

- projection- A callback function that receives the row object with table aliases and returns the selected fields object
[Basic Selects](#basic-selects)

Select specific fields from your data:
ts

```
const userNames = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      email: user.email,
    }))
)

/*
Result type: { id: number, name: string, email: string }

```ts
for (const row of userNames) {
  console.log(row.name)
}
```
*/

```

```
const userNames = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      email: user.email,
    }))
)

/*
Result type: { id: number, name: string, email: string }

```ts
for (const row of userNames) {
  console.log(row.name)
}
```
*/

```[Field Renaming](#field-renaming)

Rename fields in your results:
ts

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      userId: user.id,
      fullName: user.name,
      contactEmail: user.email,
    }))
)

```

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      userId: user.id,
      fullName: user.name,
      contactEmail: user.email,
    }))
)

```[Computed Fields](#computed-fields)

Create computed fields using expressions:
ts

```
import { gt, length } from '@tanstack/db'

const userStats = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      isAdult: gt(user.age, 18),
      nameLength: length(user.name),
    }))
)

```

```
import { gt, length } from '@tanstack/db'

const userStats = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      isAdult: gt(user.age, 18),
      nameLength: length(user.name),
    }))
)

```[Using Functions and Including All Fields](#using-functions-and-including-all-fields)

Transform your data using built-in functions:
ts

```
import { concat, upper, gt } from '@tanstack/db'

const formattedUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .select(({ user }) => ({
        ...user, // Include all user fields
        displayName: upper(concat(user.firstName, ' ', user.lastName)),
        isAdult: gt(user.age, 18),
      }))
}))

/*
Result type:
{
  id: number,
  name: string,
  email: string,
  displayName: string,
  isAdult: boolean,
}
*/

```

```
import { concat, upper, gt } from '@tanstack/db'

const formattedUsers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .select(({ user }) => ({
        ...user, // Include all user fields
        displayName: upper(concat(user.firstName, ' ', user.lastName)),
        isAdult: gt(user.age, 18),
      }))
}))

/*
Result type:
{
  id: number,
  name: string,
  email: string,
  displayName: string,
  isAdult: boolean,
}
*/

```

For a complete list of available functions, see the[Expression Functions Reference](#expression-functions-reference)section.
[Joins](#joins)

Usejointo combine data from multiple collections. Joins default toleftjoin type and only support equality conditions.

Joins in TanStack DB are a way to combine data from multiple collections, and are conceptually very similar to SQL joins. When two collections are joined, the result is a new collection that contains the combined data as single rows. The new collection is a live query collection, and will automatically update when the underlying data changes.

Ajoinwithout aselectwill return row objects that are namespaced with the aliases of the joined collections.

The result type of a join will take into account the join type, with the optionality of the joined fields being determined by the join type.
Note

We are working on anincludesystem that will enable joins that project to a hierarchical object. For example anissuerow could have acommentsproperty that is an array ofcommentrows.
See[this issue](https://github.com/TanStack/db/issues/288)for more details.[Method Signature](#method-signature-3)ts

```
join(
  { [alias]: Collection | Query },
  condition: (row: TRow) => Expression<boolean>, // Must be an `eq` condition
  joinType?: 'left' | 'right' | 'inner' | 'full'
): Query

```

```
join(
  { [alias]: Collection | Query },
  condition: (row: TRow) => Expression<boolean>, // Must be an `eq` condition
  joinType?: 'left' | 'right' | 'inner' | 'full'
): Query

```

**Parameters:**

- aliases- An object where keys are alias names and values are collections or subqueries to join
- condition- A callback function that receives the combined row object and returns an equality condition
- joinType- Optional join type:'left'(default),'right','inner', or'full'
[Basic Joins](#basic-joins)

Join users with their posts:
ts

```
const userPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
)

/*
Result type: 
{ 
  user: User,
  post?: Post, // post is optional because it is a left join
}

```ts
for (const row of userPosts) {
  console.log(row.user.name, row.post?.title)
}
```
*/

```

```
const userPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
)

/*
Result type: 
{ 
  user: User,
  post?: Post, // post is optional because it is a left join
}

```ts
for (const row of userPosts) {
  console.log(row.user.name, row.post?.title)
}
```
*/

```[Join Types](#join-types)

Specify the join type as the third parameter:
ts

```
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
      'inner', // `inner`, `left`, `right` or `full`
    )
)

```

```
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
      'inner', // `inner`, `left`, `right` or `full`
    )
)

```

Or using the aliasesleftJoin,rightJoin,innerJoinandfullJoinmethods:
[Left Join](#left-join)ts

```
// Left join - all users, even without posts
const allUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .leftJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user: User,
  post?: Post, // post is optional because it is a left join
}
*/

```

```
// Left join - all users, even without posts
const allUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .leftJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user: User,
  post?: Post, // post is optional because it is a left join
}
*/

```[Right Join](#right-join)ts

```
// Right join - all posts, even without users
const allPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .rightJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user?: User, // user is optional because it is a right join
  post: Post,
}
*/

```

```
// Right join - all posts, even without users
const allPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .rightJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user?: User, // user is optional because it is a right join
  post: Post,
}
*/

```[Inner Join](#inner-join)ts

```
// Inner join - only matching records
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .innerJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user: User,
  post: Post,
}
*/

```

```
// Inner join - only matching records
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .innerJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user: User,
  post: Post,
}
*/

```[Full Join](#full-join)ts

```
// Full join - all users and all posts
const allUsersAndPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fullJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user?: User, // user is optional because it is a full join
  post?: Post, // post is optional because it is a full join
}
*/

```

```
// Full join - all users and all posts
const allUsersAndPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fullJoin(
      { post: postsCollection }, 
      ({ user, post }) => eq(user.id, post.userId),
    )
)

/*
Result type:
{
  user?: User, // user is optional because it is a full join
  post?: Post, // post is optional because it is a full join
}
*/

```[Multiple Joins](#multiple-joins)

Chain multiple joins in a single query:
ts

```
const userPostComments = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
    .join({ comment: commentsCollection }, ({ post, comment }) => 
      eq(post.id, comment.postId)
    )
    .select(({ user, post, comment }) => ({
      userName: user.name,
      postTitle: post.title,
      commentText: comment.text,
    }))
)

```

```
const userPostComments = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
    .join({ comment: commentsCollection }, ({ post, comment }) => 
      eq(post.id, comment.postId)
    )
    .select(({ user, post, comment }) => ({
      userName: user.name,
      postTitle: post.title,
      commentText: comment.text,
    }))
)

```[Subqueries](#subqueries)

Subqueries allow you to use the result of one query as input to another, they are embedded within the query itself and are compile to a single query pipeline. They are very similar to SQL subqueries that are executed as part of a single operation.

Note that subqueries are not the same as using a live query result in afromorjoinclause in a new query. When you do that the intermediate result is fully computed and accessible to you, subqueries are internal to their parent query and not materialised to a collection themselves and so are more efficient.

See the[Caching Intermediate Results](#caching-intermediate-results)section for more details on using live query results in afromorjoinclause in a new query.
[Subqueries in from Clauses](#subqueries-in-from-clauses)

Use a subquery as the main source:
ts

```
const activeUserPosts = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery first
    const activeUsers = q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
    
    // Use the subquery in the main query
    return q
      .from({ activeUser: activeUsers })
      .join({ post: postsCollection }, ({ activeUser, post }) => 
        eq(activeUser.id, post.userId)
      )
  }
}))

```

```
const activeUserPosts = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery first
    const activeUsers = q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
    
    // Use the subquery in the main query
    return q
      .from({ activeUser: activeUsers })
      .join({ post: postsCollection }, ({ activeUser, post }) => 
        eq(activeUser.id, post.userId)
      )
  }
}))

```[Subqueries in join Clauses](#subqueries-in-join-clauses)

Join with a subquery result:
ts

```
const userRecentPosts = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery first
    const recentPosts = q
      .from({ post: postsCollection })
      .where(({ post }) => gt(post.createdAt, '2024-01-01'))
      .orderBy(({ post }) => post.createdAt, 'desc')
      .limit(1)
    
    // Use the subquery in the main query
    return q
      .from({ user: usersCollection })
      .join({ recentPost: recentPosts }, ({ user, recentPost }) => 
        eq(user.id, recentPost.userId)
      )
  }
}))

```

```
const userRecentPosts = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery first
    const recentPosts = q
      .from({ post: postsCollection })
      .where(({ post }) => gt(post.createdAt, '2024-01-01'))
      .orderBy(({ post }) => post.createdAt, 'desc')
      .limit(1)
    
    // Use the subquery in the main query
    return q
      .from({ user: usersCollection })
      .join({ recentPost: recentPosts }, ({ user, recentPost }) => 
        eq(user.id, recentPost.userId)
      )
  }
}))

```[Subquery deduplication](#subquery-deduplication)

When the same subquery is used multiple times within a query, it's automatically deduplicated and executed only once:
ts

```
const complexQuery = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery once
    const activeUsers = q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
    
    // Use the same subquery multiple times
    return q
      .from({ activeUser: activeUsers })
      .join({ post: postsCollection }, ({ activeUser, post }) => 
        eq(activeUser.id, post.userId)
      )
      .join({ comment: commentsCollection }, ({ activeUser, comment }) => 
        eq(activeUser.id, comment.userId)
      )
  }
}))

```

```
const complexQuery = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the subquery once
    const activeUsers = q
      .from({ user: usersCollection })
      .where(({ user }) => eq(user.active, true))
    
    // Use the same subquery multiple times
    return q
      .from({ activeUser: activeUsers })
      .join({ post: postsCollection }, ({ activeUser, post }) => 
        eq(activeUser.id, post.userId)
      )
      .join({ comment: commentsCollection }, ({ activeUser, comment }) => 
        eq(activeUser.id, comment.userId)
      )
  }
}))

```

In this example, theactiveUserssubquery is used twice but executed only once, improving performance.
[Complex Nested Subqueries](#complex-nested-subqueries)

Build complex queries with multiple levels of nesting:
ts

```
import { count } from '@tanstack/db'

const topUsers = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the post count subquery
    const postCounts = q
      .from({ post: postsCollection })
      .groupBy(({ post }) => post.userId)
      .select(({ post }) => ({
        userId: post.userId,
        count: count(post.id),
      }))
    
    // Build the user stats subquery
    const userStats = q
      .from({ user: usersCollection })
      .join({ postCount: postCounts }, ({ user, postCount }) => 
        eq(user.id, postCount.userId)
      )
      .select(({ user, postCount }) => ({
        id: user.id,
        name: user.name,
        postCount: postCount.count,
      }))
      .orderBy(({ userStats }) => userStats.postCount, 'desc')
      .limit(10)
    
    // Use the user stats subquery in the main query
    return q.from({ userStats })
  }
}))

```

```
import { count } from '@tanstack/db'

const topUsers = createCollection(liveQueryCollectionOptions({
  query: (q) => {
    // Build the post count subquery
    const postCounts = q
      .from({ post: postsCollection })
      .groupBy(({ post }) => post.userId)
      .select(({ post }) => ({
        userId: post.userId,
        count: count(post.id),
      }))
    
    // Build the user stats subquery
    const userStats = q
      .from({ user: usersCollection })
      .join({ postCount: postCounts }, ({ user, postCount }) => 
        eq(user.id, postCount.userId)
      )
      .select(({ user, postCount }) => ({
        id: user.id,
        name: user.name,
        postCount: postCount.count,
      }))
      .orderBy(({ userStats }) => userStats.postCount, 'desc')
      .limit(10)
    
    // Use the user stats subquery in the main query
    return q.from({ userStats })
  }
}))

```[groupBy and Aggregations](#groupby-and-aggregations)

UsegroupByto group your data and apply aggregate functions. When you use aggregates inselectwithoutgroupBy, the entire result set is treated as a single group.
[Method Signature](#method-signature-4)ts

```
groupBy(
  grouper: (row: TRow) => Expression | Expression[]
): Query

```

```
groupBy(
  grouper: (row: TRow) => Expression | Expression[]
): Query

```

**Parameters:**

- grouper- A callback function that receives the row object and returns the grouping key(s). Can return a single value or an array for multi-column grouping
[Basic Grouping](#basic-grouping)

Group users by their department and count them:
ts

```
import { count, avg } from '@tanstack/db'

const departmentStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => user.departmentId)
      .select(({ user }) => ({
        departmentId: user.departmentId,
        userCount: count(user.id),
        avgAge: avg(user.age),
      }))
}))

```

```
import { count, avg } from '@tanstack/db'

const departmentStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => user.departmentId)
      .select(({ user }) => ({
        departmentId: user.departmentId,
        userCount: count(user.id),
        avgAge: avg(user.age),
      }))
}))

```Note

IngroupByqueries, the properties in yourselectclause must either be:

- An aggregate function (likecount,sum,avg)
- A property that was used in thegroupByclause

You cannot select properties that are neither aggregated nor grouped.[Multiple Column Grouping](#multiple-column-grouping)

Group by multiple columns by returning an array from the callback:
ts

```
const userStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => [user.departmentId, user.role])
      .select(({ user }) => ({
        departmentId: user.departmentId,
        role: user.role,
        count: count(user.id),
        avgSalary: avg(user.salary),
      }))
}))

```

```
const userStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => [user.departmentId, user.role])
      .select(({ user }) => ({
        departmentId: user.departmentId,
        role: user.role,
        count: count(user.id),
        avgSalary: avg(user.salary),
      }))
}))

```[Aggregate Functions](#aggregate-functions)

Use various aggregate functions to summarize your data:
ts

```
import { count, sum, avg, min, max } from '@tanstack/db'

const orderStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ order: ordersCollection })
      .groupBy(({ order }) => order.customerId)
      .select(({ order }) => ({
        customerId: order.customerId,
        totalOrders: count(order.id),
        totalAmount: sum(order.amount),
        avgOrderValue: avg(order.amount),
        minOrder: min(order.amount),
        maxOrder: max(order.amount),
      }))
}))

```

```
import { count, sum, avg, min, max } from '@tanstack/db'

const orderStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ order: ordersCollection })
      .groupBy(({ order }) => order.customerId)
      .select(({ order }) => ({
        customerId: order.customerId,
        totalOrders: count(order.id),
        totalAmount: sum(order.amount),
        avgOrderValue: avg(order.amount),
        minOrder: min(order.amount),
        maxOrder: max(order.amount),
      }))
}))

```

See the[Aggregate Functions](#aggregate-functions)section for a complete list of available aggregate functions.
[Having Clauses](#having-clauses)

Filter aggregated results usinghaving- this is similar to thewhereclause, but is applied after the aggregation has been performed.
[Method Signature](#method-signature-5)ts

```
having(
  condition: (row: TRow) => Expression<boolean>
): Query

```

```
having(
  condition: (row: TRow) => Expression<boolean>
): Query

```

**Parameters:**

- condition- A callback function that receives the aggregated row object and returns a boolean expression
ts

```
const highValueCustomers = createLiveQueryCollection((q) =>
  q
    .from({ order: ordersCollection })
    .groupBy(({ order }) => order.customerId)
    .select(({ order }) => ({
      customerId: order.customerId,
      totalSpent: sum(order.amount),
      orderCount: count(order.id),
    }))
    .having(({ order }) => gt(sum(order.amount), 1000))
)

```

```
const highValueCustomers = createLiveQueryCollection((q) =>
  q
    .from({ order: ordersCollection })
    .groupBy(({ order }) => order.customerId)
    .select(({ order }) => ({
      customerId: order.customerId,
      totalSpent: sum(order.amount),
      orderCount: count(order.id),
    }))
    .having(({ order }) => gt(sum(order.amount), 1000))
)

```[Implicit Single-Group Aggregation](#implicit-single-group-aggregation)

When you use aggregates withoutgroupBy, the entire result set is grouped:
ts

```
const overallStats = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      totalUsers: count(user.id),
      avgAge: avg(user.age),
      maxSalary: max(user.salary),
    }))
)

```

```
const overallStats = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .select(({ user }) => ({
      totalUsers: count(user.id),
      avgAge: avg(user.age),
      maxSalary: max(user.salary),
    }))
)

```

This is equivalent to grouping the entire collection into a single group.
[Accessing Grouped Data](#accessing-grouped-data)

Grouped results can be accessed by the group key:
ts

```
const deptStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => user.departmentId)
      .select(({ user }) => ({
        departmentId: user.departmentId,
        count: count(user.id),
      }))
}))

// Access by department ID
const engineeringStats = deptStats.get(1)

```

```
const deptStats = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .groupBy(({ user }) => user.departmentId)
      .select(({ user }) => ({
        departmentId: user.departmentId,
        count: count(user.id),
      }))
}))

// Access by department ID
const engineeringStats = deptStats.get(1)

```

> **Note**: Grouped results are keyed differently based on the grouping:
> - **Single column grouping**: Keyed by the actual value (e.g.,deptStats.get(1))
> - **Multiple column grouping**: Keyed by a JSON string of the grouped values (e.g.,userStats.get('[1,"admin"]'))

[Order By, Limit, and Offset](#order-by-limit-and-offset)

UseorderBy,limit, andoffsetto control the order and pagination of your results. Ordering is performed incrementally for optimal performance.
[Method Signatures](#method-signatures)ts

```
orderBy(
  selector: (row: TRow) => Expression,
  direction?: 'asc' | 'desc'
): Query

limit(count: number): Query

offset(count: number): Query

```

```
orderBy(
  selector: (row: TRow) => Expression,
  direction?: 'asc' | 'desc'
): Query

limit(count: number): Query

offset(count: number): Query

```

**Parameters:**

- selector- A callback function that receives the row object and returns the value to sort by
- direction- Sort direction:'asc'(default) or'desc'
- count- Number of rows to limit or skip
[Basic Ordering](#basic-ordering)

Sort results by a single column:
ts

```
const sortedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.name)
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```

```
const sortedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.name)
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```[Multiple Column Ordering](#multiple-column-ordering)

Order by multiple columns:
ts

```
const sortedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.departmentId, 'asc')
    .orderBy(({ user }) => user.name, 'asc')
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      departmentId: user.departmentId,
    }))
)

```

```
const sortedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.departmentId, 'asc')
    .orderBy(({ user }) => user.name, 'asc')
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
      departmentId: user.departmentId,
    }))
)

```[Descending Order](#descending-order)

Usedescfor descending order:
ts

```
const recentPosts = createLiveQueryCollection((q) =>
  q
    .from({ post: postsCollection })
    .orderBy(({ post }) => post.createdAt, 'desc')
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      createdAt: post.createdAt,
    }))
)

```

```
const recentPosts = createLiveQueryCollection((q) =>
  q
    .from({ post: postsCollection })
    .orderBy(({ post }) => post.createdAt, 'desc')
    .select(({ post }) => ({
      id: post.id,
      title: post.title,
      createdAt: post.createdAt,
    }))
)

```[Pagination with limit and offset](#pagination-with-limit-and-offset)

Skip results usingoffset:
ts

```
const page2Users = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.name, 'asc')
    .limit(20)
    .offset(20) // Skip first 20 results
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```

```
const page2Users = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .orderBy(({ user }) => user.name, 'asc')
    .limit(20)
    .offset(20) // Skip first 20 results
    .select(({ user }) => ({
      id: user.id,
      name: user.name,
    }))
)

```[Composable Queries](#composable-queries)

Build complex queries by composing smaller, reusable parts. This approach makes your queries more maintainable and allows for better performance through caching.
[Conditional Query Building](#conditional-query-building)

Build queries based on runtime conditions:
ts

```
import { Query, eq } from '@tanstack/db'

function buildUserQuery(options: { activeOnly?: boolean; limit?: number }) {
  let query = new Query().from({ user: usersCollection })
  
  if (options.activeOnly) {
    query = query.where(({ user }) => eq(user.active, true))
  }
  
  if (options.limit) {
    query = query.limit(options.limit)
  }
  
  return query.select(({ user }) => ({
    id: user.id,
    name: user.name,
  }))
}

const activeUsers = createLiveQueryCollection(buildUserQuery({ activeOnly: true, limit: 10 }))

```

```
import { Query, eq } from '@tanstack/db'

function buildUserQuery(options: { activeOnly?: boolean; limit?: number }) {
  let query = new Query().from({ user: usersCollection })
  
  if (options.activeOnly) {
    query = query.where(({ user }) => eq(user.active, true))
  }
  
  if (options.limit) {
    query = query.limit(options.limit)
  }
  
  return query.select(({ user }) => ({
    id: user.id,
    name: user.name,
  }))
}

const activeUsers = createLiveQueryCollection(buildUserQuery({ activeOnly: true, limit: 10 }))

```[Caching Intermediate Results](#caching-intermediate-results)

The result of a live query collection is a collection itself, and will automatically update when the underlying data changes. This means that you can use the result of a live query collection as a source in another live query collection. This pattern is useful for building complex queries where you want to cache intermediate results to make further queries faster.
ts

```
// Base query for active users
const activeUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
)

// Query that depends on active users
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: activeUsers })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
    .select(({ user, post }) => ({
      userName: user.name,
      postTitle: post.title,
    }))
)

```

```
// Base query for active users
const activeUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .where(({ user }) => eq(user.active, true))
)

// Query that depends on active users
const activeUserPosts = createLiveQueryCollection((q) =>
  q
    .from({ user: activeUsers })
    .join({ post: postsCollection }, ({ user, post }) => 
      eq(user.id, post.userId)
    )
    .select(({ user, post }) => ({
      userName: user.name,
      postTitle: post.title,
    }))
)

```[Reusable Query Definitions](#reusable-query-definitions)

You can use theQueryclass to create reusable query definitions. This is useful for building complex queries where you want to reuse the same query builder instance multiple times throughout your application.
ts

```
import { Query, eq } from '@tanstack/db'

// Create a reusable query builder
const userQuery = new Query()
  .from({ user: usersCollection })
  .where(({ user }) => eq(user.active, true))

// Use it in different contexts
const activeUsers = createLiveQueryCollection({
  query: userQuery.select(({ user }) => ({
    id: user.id,
    name: user.name,
  }))
})

// Or as a subquery
const userPosts = createLiveQueryCollection((q) =>
  q
    .from({ activeUser: userQuery })
    .join({ post: postsCollection }, ({ activeUser, post }) => 
      eq(activeUser.id, post.userId)
    )
)

```

```
import { Query, eq } from '@tanstack/db'

// Create a reusable query builder
const userQuery = new Query()
  .from({ user: usersCollection })
  .where(({ user }) => eq(user.active, true))

// Use it in different contexts
const activeUsers = createLiveQueryCollection({
  query: userQuery.select(({ user }) => ({
    id: user.id,
    name: user.name,
  }))
})

// Or as a subquery
const userPosts = createLiveQueryCollection((q) =>
  q
    .from({ activeUser: userQuery })
    .join({ post: postsCollection }, ({ activeUser, post }) => 
      eq(activeUser.id, post.userId)
    )
)

```[Reusable Callback Functions](#reusable-callback-functions)

UseRef<MyType>to create reusable callback functions:
ts

```
import { Ref, eq, gt, and } from '@tanstack/db'

// Create reusable callbacks
const isActiveUser = (user: Ref<User>) => eq(user.active, true)
const isAdultUser = (user: Ref<User>) => gt(user.age, 18)

// Use them in queries
const activeAdults = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => and(isActiveUser(user), isAdultUser(user)))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
        age: user.age,
      }))
}))

```

```
import { Ref, eq, gt, and } from '@tanstack/db'

// Create reusable callbacks
const isActiveUser = (user: Ref<User>) => eq(user.active, true)
const isAdultUser = (user: Ref<User>) => gt(user.age, 18)

// Use them in queries
const activeAdults = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .where(({ user }) => and(isActiveUser(user), isAdultUser(user)))
      .select(({ user }) => ({
        id: user.id,
        name: user.name,
        age: user.age,
      }))
}))

```

You can also create callbacks that take the whole row and pass them directly towhere:
ts

```
// Callback that takes the whole row
const isHighValueCustomer = (row: { user: User; order: Order }) => 
  row.user.active && row.order.amount > 1000

// Use directly in where clause
const highValueCustomers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .join({ order: ordersCollection }, ({ user, order }) => 
        eq(user.id, order.userId)
      )
      .where(isHighValueCustomer)
      .select(({ user, order }) => ({
        userName: user.name,
        orderAmount: order.amount,
      }))
}))

```

```
// Callback that takes the whole row
const isHighValueCustomer = (row: { user: User; order: Order }) => 
  row.user.active && row.order.amount > 1000

// Use directly in where clause
const highValueCustomers = createCollection(liveQueryCollectionOptions({
  query: (q) =>
    q
      .from({ user: usersCollection })
      .join({ order: ordersCollection }, ({ user, order }) => 
        eq(user.id, order.userId)
      )
      .where(isHighValueCustomer)
      .select(({ user, order }) => ({
        userName: user.name,
        orderAmount: order.amount,
      }))
}))

```

This approach makes your query logic more modular and testable.
[Expression Functions Reference](#expression-functions-reference)

The query system provides a comprehensive set of functions for filtering, transforming, and aggregating data.
[Comparison Operators](#comparison-operators)[eq(left, right)](#eqleft-right)

Equality comparison:
ts

```
eq(user.id, 1)
eq(user.name, 'John')

```

```
eq(user.id, 1)
eq(user.name, 'John')

```[gt(left, right), gte(left, right), lt(left, right), lte(left, right)](#gtleft-right-gteleft-right-ltleft-right-lteleft-right)

Numeric, string and date comparisons:
ts

```
gt(user.age, 18)
gte(user.salary, 50000)
lt(user.createdAt, new Date('2024-01-01'))
lte(user.rating, 5)

```

```
gt(user.age, 18)
gte(user.salary, 50000)
lt(user.createdAt, new Date('2024-01-01'))
lte(user.rating, 5)

```[inArray(value, array)](#inarrayvalue-array)

Check if a value is in an array:
ts

```
inArray(user.id, [1, 2, 3])
inArray(user.role, ['admin', 'moderator'])

```

```
inArray(user.id, [1, 2, 3])
inArray(user.role, ['admin', 'moderator'])

```[like(value, pattern), ilike(value, pattern)](#likevalue-pattern-ilikevalue-pattern)

String pattern matching:
ts

```
like(user.name, 'John%')    // Case-sensitive
ilike(user.email, '%@gmail.com')  // Case-insensitive

```

```
like(user.name, 'John%')    // Case-sensitive
ilike(user.email, '%@gmail.com')  // Case-insensitive

```[Logical Operators](#logical-operators)[and(...conditions)](#andconditions)

Combine conditions with AND logic:
ts

```
and(
  eq(user.active, true),
  gt(user.age, 18),
  eq(user.role, 'user')
)

```

```
and(
  eq(user.active, true),
  gt(user.age, 18),
  eq(user.role, 'user')
)

```[or(...conditions)](#orconditions)

Combine conditions with OR logic:
ts

```
or(
  eq(user.role, 'admin'),
  eq(user.role, 'moderator')
)

```

```
or(
  eq(user.role, 'admin'),
  eq(user.role, 'moderator')
)

```[not(condition)](#notcondition)

Negate a condition:
ts

```
not(eq(user.active, false))

```

```
not(eq(user.active, false))

```[String Functions](#string-functions)[upper(value), lower(value)](#uppervalue-lowervalue)

Convert case:
ts

```
upper(user.name)  // 'JOHN'
lower(user.email) // 'john@example.com'

```

```
upper(user.name)  // 'JOHN'
lower(user.email) // 'john@example.com'

```[length(value)](#lengthvalue)

Get string or array length:
ts

```
length(user.name)     // String length
length(user.tags)     // Array length

```

```
length(user.name)     // String length
length(user.tags)     // Array length

```[concat(...values)](#concatvalues)

Concatenate strings:
ts

```
concat(user.firstName, ' ', user.lastName)
concat('User: ', user.name, ' (', user.id, ')')

```

```
concat(user.firstName, ' ', user.lastName)
concat('User: ', user.name, ' (', user.id, ')')

```[Mathematical Functions](#mathematical-functions)[add(left, right)](#addleft-right)

Add two numbers:
ts

```
add(user.salary, user.bonus)

```

```
add(user.salary, user.bonus)

```[coalesce(...values)](#coalescevalues)

Return the first non-null value:
ts

```
coalesce(user.displayName, user.name, 'Unknown')

```

```
coalesce(user.displayName, user.name, 'Unknown')

```[Aggregate Functions](#aggregate-functions-1)[count(value)](#countvalue)

Count non-null values:
ts

```
count(user.id)        // Count all users
count(user.postId)    // Count users with posts

```

```
count(user.id)        // Count all users
count(user.postId)    // Count users with posts

```[sum(value)](#sumvalue)

Sum numeric values:
ts

```
sum(order.amount)
sum(user.salary)

```

```
sum(order.amount)
sum(user.salary)

```[avg(value)](#avgvalue)

Calculate average:
ts

```
avg(user.salary)
avg(order.amount)

```

```
avg(user.salary)
avg(order.amount)

```[min(value), max(value)](#minvalue-maxvalue)

Find minimum and maximum values:
ts

```
min(user.salary)
max(order.amount)

```

```
min(user.salary)
max(order.amount)

```[Function Composition](#function-composition)

Functions can be composed and chained:
ts

```
// Complex condition
and(
  eq(user.active, true),
  or(
    gt(user.age, 25),
    eq(user.role, 'admin')
  ),
  not(inArray(user.id, bannedUserIds))
)

// Complex transformation
concat(
  upper(user.firstName),
  ' ',
  upper(user.lastName),
  ' (',
  user.id,
  ')'
)

// Complex aggregation
avg(add(user.salary, coalesce(user.bonus, 0)))

```

```
// Complex condition
and(
  eq(user.active, true),
  or(
    gt(user.age, 25),
    eq(user.role, 'admin')
  ),
  not(inArray(user.id, bannedUserIds))
)

// Complex transformation
concat(
  upper(user.firstName),
  ' ',
  upper(user.lastName),
  ' (',
  user.id,
  ')'
)

// Complex aggregation
avg(add(user.salary, coalesce(user.bonus, 0)))

```[Functional Variants](#functional-variants)

The functional variant API provides an alternative to the standard API, offering more flexibility for complex transformations. With functional variants, the callback functions contain actual code that gets executed to perform the operation, giving you the full power of JavaScript at your disposal.
Warning

The functional variant API cannot be optimized by the query optimizer or use collection indexes. It is intended for use in rare cases where the standard API is not sufficient.[Functional Select](#functional-select)

Usefn.select()for complex transformations with JavaScript logic:
ts

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row) => ({
      id: row.user.id,
      displayName: `${row.user.firstName} ${row.user.lastName}`,
      salaryTier: row.user.salary > 100000 ? 'senior' : 'junior',
      emailDomain: row.user.email.split('@')[1],
      isHighEarner: row.user.salary > 75000,
    }))
)

```

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row) => ({
      id: row.user.id,
      displayName: `${row.user.firstName} ${row.user.lastName}`,
      salaryTier: row.user.salary > 100000 ? 'senior' : 'junior',
      emailDomain: row.user.email.split('@')[1],
      isHighEarner: row.user.salary > 75000,
    }))
)

```[Functional Where](#functional-where)

Usefn.where()for complex filtering logic:
ts

```
const specialUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.where((row) => {
      const user = row.user
      return user.active && 
             (user.age > 25 || user.role === 'admin') &&
             user.email.includes('@company.com')
    })
)

```

```
const specialUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.where((row) => {
      const user = row.user
      return user.active && 
             (user.age > 25 || user.role === 'admin') &&
             user.email.includes('@company.com')
    })
)

```[Functional Having](#functional-having)

Usefn.having()for complex aggregation filtering:
ts

```
const highValueCustomers = createLiveQueryCollection((q) =>
  q
    .from({ order: ordersCollection })
    .groupBy(({ order }) => order.customerId)
    .select(({ order }) => ({
      customerId: order.customerId,
      totalSpent: sum(order.amount),
      orderCount: count(order.id),
    }))
    .fn.having((row) => {
      return row.totalSpent > 1000 && row.orderCount >= 3
    })
)

```

```
const highValueCustomers = createLiveQueryCollection((q) =>
  q
    .from({ order: ordersCollection })
    .groupBy(({ order }) => order.customerId)
    .select(({ order }) => ({
      customerId: order.customerId,
      totalSpent: sum(order.amount),
      orderCount: count(order.id),
    }))
    .fn.having((row) => {
      return row.totalSpent > 1000 && row.orderCount >= 3
    })
)

```[Complex Transformations](#complex-transformations)

Functional variants excel at complex data transformations:
ts

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row) => {
      const user = row.user
      const fullName = `${user.firstName} ${user.lastName}`.trim()
      const emailDomain = user.email.split('@')[1]
      const ageGroup = user.age < 25 ? 'young' : user.age < 50 ? 'adult' : 'senior'
      
      return {
        userId: user.id,
        displayName: fullName || user.name,
        contactInfo: {
          email: user.email,
          domain: emailDomain,
          isCompanyEmail: emailDomain === 'company.com'
        },
        demographics: {
          age: user.age,
          ageGroup: ageGroup,
          isAdult: user.age >= 18
        },
        status: user.active ? 'active' : 'inactive',
        profileStrength: fullName && user.email && user.age ? 'complete' : 'incomplete'
      }
    })
)

```

```
const userProfiles = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row) => {
      const user = row.user
      const fullName = `${user.firstName} ${user.lastName}`.trim()
      const emailDomain = user.email.split('@')[1]
      const ageGroup = user.age < 25 ? 'young' : user.age < 50 ? 'adult' : 'senior'
      
      return {
        userId: user.id,
        displayName: fullName || user.name,
        contactInfo: {
          email: user.email,
          domain: emailDomain,
          isCompanyEmail: emailDomain === 'company.com'
        },
        demographics: {
          age: user.age,
          ageGroup: ageGroup,
          isAdult: user.age >= 18
        },
        status: user.active ? 'active' : 'inactive',
        profileStrength: fullName && user.email && user.age ? 'complete' : 'incomplete'
      }
    })
)

```[Type Inference](#type-inference)

Functional variants maintain full TypeScript support:
ts

```
const processedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row): ProcessedUser => ({
      id: row.user.id,
      name: row.user.name.toUpperCase(),
      age: row.user.age,
      ageGroup: row.user.age < 25 ? 'young' : row.user.age < 50 ? 'adult' : 'senior',
    }))
)

```

```
const processedUsers = createLiveQueryCollection((q) =>
  q
    .from({ user: usersCollection })
    .fn.select((row): ProcessedUser => ({
      id: row.user.id,
      name: row.user.name.toUpperCase(),
      age: row.user.age,
      ageGroup: row.user.age < 25 ? 'young' : row.user.age < 50 ? 'adult' : 'senior',
    }))
)

```[When to Use Functional Variants](#when-to-use-functional-variants)

Use functional variants when you need:

- Complex JavaScript logic that can't be expressed with built-in functions
- Integration with external libraries or utilities
- Full JavaScript power for custom operations

The callbacks in functional variants are actual JavaScript functions that get executed, unlike the standard API which uses declarative expressions. This gives you complete control over the logic but comes with the trade-off of reduced optimization opportunities.

However, prefer the standard API when possible, as it provides better performance and optimization opportunities.[Edit on GitHub](https://github.com/tanstack/db/edit/main/docs/guides/live-queries.md)[React Adapter](/db/latest/docs/framework/react/adapter)[Error Handling](/db/latest/docs/guides/error-handling)Our Partners###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.###### Subscribe to Bytes

Your weekly dose of JavaScript news. Delivered every Monday to over 100,000 devs, for free.SubscribeNo spam. Unsubscribe at any time.<iframe src="https://www.googletagmanager.com/ns.html?id=GTM-5N57KQT4" height="0" width="0" style="display:none;visibility:hidden" title="gtm"></iframe>