// Package vercel implements the direct-targeting subset of Vercel Flags as an
// OpenFeature provider.
//
// Compatibility is pinned to @vercel/flags-core 1.9.0 source commit
// 45e1bb20aafc8d189e870bb205d02421561ce683 and was cross-checked against the
// repository's 1.4.0 datafile shapes. The datafile protocol is undocumented.
// This provider supports boolean variants, direct string targets, paused numeric
// outcomes, and numeric fallthrough outcomes. Rules, splits, experiments,
// environment reuse, and unknown active configuration fields are rejected per flag.
package vercel
