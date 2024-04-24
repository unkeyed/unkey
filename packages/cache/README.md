<div align="center">
    <h1 align="center">@unkey/cache</h1>
    <h5>Cache all the things</h5>
</div>

<div align="center">
  <a href="https://unkey.dev">unkey.dev</a>
</div>
<br/>






Battle tested, strongly typed caching with metrics and tracing out of the box.

## Features

- Tiered caching
- Memory Cache
- Cloudflare Zone Cache
- Cloudflare KV cache (maybe)
- Upstash Redis cache (maybe)
- Metrics (axiom)
- Tracing (axiom)



## Glossary

### Store

Stores are the low-level storage engines that the cache uses to store and retrieve data.
Any persistent key-value store, implementing the `Store` interface can be used.

Note: The store should handle expiration on its own, removing old data is out of scope of this package.

