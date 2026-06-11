//! Port of pkg/cache: an in-memory stale-while-revalidate cache (sync,
//! std-only — no async runtime).
//!
//! Semantics mirror the Go implementation as used by frontline:
//!   - entries younger than `fresh` are returned directly
//!   - entries older than `fresh` but younger than `stale` are returned
//!     immediately while a background thread revalidates them
//!   - entries older than `stale` (or missing) are loaded inline
//!   - "not found" results are cached as nulls (cache.Null) so hot misses
//!     don't hammer the database (caches.DefaultFindFirstOp)
//!   - loader errors are never cached (cache.Noop)

use std::collections::{HashMap, HashSet};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::error::FrontlineError;

struct Entry<V> {
    /// None is a cached null: the origin authoritatively said "not found".
    value: Option<V>,
    inserted: Instant,
}

pub struct SwrCache<V> {
    name: &'static str,
    fresh: Duration,
    stale: Duration,
    max_size: usize,
    entries: Mutex<HashMap<String, Entry<V>>>,
    revalidating: Mutex<HashSet<String>>,
}

impl<V: Clone + Send + Sync + 'static> SwrCache<V> {
    pub fn new(name: &'static str, fresh: Duration, stale: Duration, max_size: usize) -> Arc<Self> {
        Arc::new(Self {
            name,
            fresh,
            stale,
            max_size,
            entries: Mutex::new(HashMap::new()),
            revalidating: Mutex::new(HashSet::new()),
        })
    }

    fn insert(&self, key: String, value: Option<V>) {
        let mut entries = self.entries.lock().unwrap();
        if entries.len() >= self.max_size && !entries.contains_key(&key) {
            // Evict expired entries first; if everything is still live, drop
            // an arbitrary entry. The Go cache is LRU — this is a simpler
            // policy with the same bound on memory.
            let now = Instant::now();
            let stale = self.stale;
            entries.retain(|_, e| now.duration_since(e.inserted) < stale);
            if entries.len() >= self.max_size {
                if let Some(victim) = entries.keys().next().cloned() {
                    entries.remove(&victim);
                }
            }
        }
        entries.insert(
            key,
            Entry {
                value,
                inserted: Instant::now(),
            },
        );
    }

    /// Stale-while-revalidate read-through. The loader returns
    /// `Ok(Some(v))` on success, `Ok(None)` when the origin says not-found
    /// (cached as null), and `Err` on transient failure (not cached).
    pub fn swr(
        self: &Arc<Self>,
        key: &str,
        loader: impl FnOnce() -> Result<Option<V>, FrontlineError> + Send + 'static,
    ) -> Result<Option<V>, FrontlineError> {
        enum State<V> {
            FreshHit(Option<V>),
            StaleHit(Option<V>),
            Miss,
        }

        let state = {
            let entries = self.entries.lock().unwrap();
            match entries.get(key) {
                Some(e) => {
                    let age = Instant::now().duration_since(e.inserted);
                    if age < self.fresh {
                        State::FreshHit(e.value.clone())
                    } else if age < self.stale {
                        State::StaleHit(e.value.clone())
                    } else {
                        State::Miss
                    }
                }
                None => State::Miss,
            }
        };

        match state {
            State::FreshHit(v) => Ok(v),
            State::StaleHit(v) => {
                // Serve stale, revalidate in the background. Deduplicate so a
                // hot key triggers one refresh, not one per request.
                if self.revalidating.lock().unwrap().insert(key.to_string()) {
                    let cache = Arc::clone(self);
                    let key = key.to_string();
                    std::thread::spawn(move || {
                        match loader() {
                            Ok(value) => cache.insert(key.clone(), value),
                            Err(err) => {
                                crate::log_debug!("background revalidation failed",
                                    "cache" => cache.name, "key" => key, "error" => err);
                            }
                        }
                        cache.revalidating.lock().unwrap().remove(&key);
                    });
                }
                Ok(v)
            }
            State::Miss => {
                let loaded = loader()?;
                self.insert(key.to_string(), loaded.clone());
                Ok(loaded)
            }
        }
    }

    /// Port of cache.SWRWithFallback as used by the cert manager: checks all
    /// candidate keys, returns the first hit, and on miss fetches from origin
    /// and caches under the canonical key returned by the loader.
    pub fn swr_with_fallback(
        self: &Arc<Self>,
        candidates: &[String],
        loader: impl FnOnce() -> Result<Option<(V, String)>, FrontlineError>,
    ) -> Result<Option<V>, FrontlineError> {
        {
            let entries = self.entries.lock().unwrap();
            let now = Instant::now();
            for key in candidates {
                if let Some(e) = entries.get(key.as_str()) {
                    if now.duration_since(e.inserted) < self.stale {
                        // Freshness-based background revalidation is skipped
                        // on the fallback path; certs are long-lived.
                        return Ok(e.value.clone());
                    }
                }
            }
        }

        match loader()? {
            Some((value, canonical_key)) => {
                self.insert(canonical_key, Some(value.clone()));
                Ok(Some(value))
            }
            None => {
                // Cache the null under the exact (first) candidate so repeated
                // lookups for an unconfigured hostname don't hit the origin.
                if let Some(first) = candidates.first() {
                    self.insert(first.clone(), None);
                }
                Ok(None)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[test]
    fn caches_values_and_nulls() {
        let cache: Arc<SwrCache<String>> =
            SwrCache::new("test", Duration::from_secs(5), Duration::from_secs(60), 10);
        let calls = Arc::new(AtomicUsize::new(0));

        let c = calls.clone();
        let v = cache
            .swr("a", move || {
                c.fetch_add(1, Ordering::SeqCst);
                Ok(Some("hello".to_string()))
            })
            .unwrap();
        assert_eq!(v.as_deref(), Some("hello"));

        // Second read is a fresh hit; loader must not run again.
        let c = calls.clone();
        let v = cache
            .swr("a", move || {
                c.fetch_add(1, Ordering::SeqCst);
                Ok(Some("changed".to_string()))
            })
            .unwrap();
        assert_eq!(v.as_deref(), Some("hello"));
        assert_eq!(calls.load(Ordering::SeqCst), 1);

        // Null caching: not-found is remembered.
        let c = calls.clone();
        let v = cache
            .swr("missing", move || {
                c.fetch_add(1, Ordering::SeqCst);
                Ok(None)
            })
            .unwrap();
        assert!(v.is_none());
        let c = calls.clone();
        let v = cache
            .swr("missing", move || {
                c.fetch_add(1, Ordering::SeqCst);
                Ok(Some("zombie".to_string()))
            })
            .unwrap();
        assert!(v.is_none());
        assert_eq!(calls.load(Ordering::SeqCst), 2);
    }

    #[test]
    fn errors_are_not_cached() {
        let cache: Arc<SwrCache<String>> =
            SwrCache::new("test", Duration::from_secs(5), Duration::from_secs(60), 10);

        let r = cache.swr("k", || {
            Err(FrontlineError::new(
                crate::error::urn::CONFIG_LOAD_FAILED,
                "boom",
                "",
            ))
        });
        assert!(r.is_err());

        let v = cache.swr("k", || Ok(Some("ok".to_string()))).unwrap();
        assert_eq!(v.as_deref(), Some("ok"));
    }

    #[test]
    fn fallback_caches_under_canonical_key() {
        let cache: Arc<SwrCache<String>> = SwrCache::new(
            "certs",
            Duration::from_secs(60),
            Duration::from_secs(120),
            10,
        );
        let candidates = vec!["api.example.com".to_string(), "*.example.com".to_string()];

        let v = cache
            .swr_with_fallback(&candidates, || {
                Ok(Some((
                    "wildcard-cert".to_string(),
                    "*.example.com".to_string(),
                )))
            })
            .unwrap();
        assert_eq!(v.as_deref(), Some("wildcard-cert"));

        // Lookup for a sibling subdomain hits the wildcard's canonical entry.
        let candidates2 = vec!["web.example.com".to_string(), "*.example.com".to_string()];
        let v = cache
            .swr_with_fallback(&candidates2, || panic!("loader must not run"))
            .unwrap();
        assert_eq!(v.as_deref(), Some("wildcard-cert"));
    }
}
