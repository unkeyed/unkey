// Package domainverify decides whether a hostname is verified, for every table
// that holds hostnames.
//
// Deploy custom domains and portal domains live in separate tables with their
// own verification workflows, but one hostname can only route one way, so the
// DNS rules and the contention rules must be identical for both. This package
// owns those rules; each workflow owns its own row reads, status writes and
// success steps.
//
// Contention spans both tables: a hostname another workspace holds verified in
// either table can only be claimed with TXT proof, and a successful claim
// revokes the other workspace's row wherever it lives.
package domainverify
