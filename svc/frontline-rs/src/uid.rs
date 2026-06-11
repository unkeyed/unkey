//! Port of pkg/uid: prefixed random identifiers, without a rand crate.
//!
//! IDs are identifiers, not secrets, so a fast non-cryptographic PRNG seeded
//! from the system clock and thread identity is sufficient.

use std::cell::Cell;
use std::time::{SystemTime, UNIX_EPOCH};

const ALPHABET: &[u8] = b"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

thread_local! {
    static STATE: Cell<u64> = Cell::new(seed());
}

fn seed() -> u64 {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos() as u64)
        .unwrap_or(0xdead_beef);
    // Mix in the address of a stack local for per-thread entropy.
    let local = 0u8;
    let addr = &local as *const u8 as u64;
    splitmix64(now ^ addr.rotate_left(32) ^ 0x9e37_79b9_7f4a_7c15)
}

fn splitmix64(mut x: u64) -> u64 {
    x = x.wrapping_add(0x9e37_79b9_7f4a_7c15);
    let mut z = x;
    z = (z ^ (z >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
    z = (z ^ (z >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
    z ^ (z >> 31)
}

/// Next pseudo-random u64 for this thread.
pub fn next_u64() -> u64 {
    STATE.with(|s| {
        let v = splitmix64(s.get());
        s.set(v);
        v
    })
}

/// Uniform-ish random index below n (n must be > 0 and small).
pub fn next_below(n: usize) -> usize {
    (next_u64() % n as u64) as usize
}

/// In-place Fisher-Yates shuffle, replacing rand.Shuffle.
pub fn shuffle<T>(items: &mut [T]) {
    if items.len() < 2 {
        return;
    }
    for i in (1..items.len()).rev() {
        let j = next_below(i + 1);
        items.swap(i, j);
    }
}

/// Generates an identifier like "req_3fJk...", matching the shape of Go's
/// uid.New(prefix).
pub fn new(prefix: &str) -> String {
    let mut body = String::with_capacity(24);
    for _ in 0..24 {
        body.push(ALPHABET[next_below(ALPHABET.len())] as char);
    }
    format!("{prefix}_{body}")
}

pub const REQUEST_PREFIX: &str = "req";
pub const INSTANCE_PREFIX: &str = "ins";

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashSet;

    #[test]
    fn ids_have_prefix_and_are_unique() {
        let mut seen = HashSet::new();
        for _ in 0..1000 {
            let id = new(REQUEST_PREFIX);
            assert!(id.starts_with("req_"));
            assert_eq!(id.len(), 4 + 24);
            assert!(seen.insert(id));
        }
    }

    #[test]
    fn shuffle_preserves_elements() {
        let mut v: Vec<u32> = (0..100).collect();
        shuffle(&mut v);
        let mut sorted = v.clone();
        sorted.sort_unstable();
        assert_eq!(sorted, (0..100).collect::<Vec<_>>());
    }
}
