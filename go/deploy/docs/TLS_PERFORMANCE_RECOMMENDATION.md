# TLS Certificate Loading Performance Recommendation

## Summary

The current implementation loads certificates from disk on every TLS connection to enable automatic certificate rotation. While this adds ~120μs (8.6%) overhead to each handshake, it provides immediate rotation detection without service restart.

## Benchmark Results

```
Certificate Loading:
- Dynamic (current): ~90μs per operation
- Cached: ~0.2ns per operation (450,000x faster)

Full TLS Handshake:
- Dynamic loading: ~1.52ms
- Static loading: ~1.40ms
- Overhead: ~120μs (8.6%)

Under Concurrent Load (32 goroutines):
- Uncached: ~30μs per operation
- Cached (5s TTL): ~68ns per operation (442x faster)
```

## Recommendation

For most production use cases, the current implementation is sufficient:

1. **8.6% overhead is acceptable** for the security benefit of immediate rotation
2. **TLS handshakes are infrequent** compared to application requests
3. **Filesystem caching** reduces impact under load

## Optional Performance Optimization

If your workload has extremely high connection rates, enable certificate caching:

```bash
# Enable 5-second certificate cache
export UNKEY_METALD_TLS_ENABLE_CERT_CACHING=true
export UNKEY_METALD_TLS_CERT_CACHE_TTL=5s
```

This provides:
- 442x performance improvement under load
- Certificate rotation detected within TTL window
- Good balance of security and performance

## When to Use Caching

Consider enabling caching if:
- Connection rate > 1000/second
- Certificate rotation frequency > 1 hour
- Performance monitoring shows TLS overhead > 10%

## SPIFFE Alternative

When SPIFFE/SPIRE is deployed, it handles certificate caching internally with:
- Automatic hourly rotation
- In-memory certificate storage
- No disk I/O on connections
- Best security + performance combination