# useUnkey Example

This example creates a GraphQL Yoga Server using node http and the following schema:

```graphql
type Query {
  hello(name: String): String!
}
```

It includes the GraphiQL Playground to test queries.

- Start server `pnpm start`
- Visit the [GraphiQL Playground](http://localhost:4000/graphql) at [http://localhost:4000/graphql](http://localhost:4000/graphql)
- In GraphiQL, try to query:

```graphql
{
  hello
}
```

Should received an error:

```json
{
  "errors": [
    {
      "message": "Unexpected error.",
      "extensions": {}
    }
  ]
}
```

as the example UnKey token of "123" is invalid and will throw:

```bash
Verifying key... 123
ERR RateLimitError: Rate limit exceeded!
```
