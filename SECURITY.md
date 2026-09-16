# Security Policy

## Scope

We accept security reports that affect the **Unkey API**(api.unkey.com) or the **Unkey dashboard** (app.unkey.com). That includes authentication, authorization, key handling, rate limiting, data exposure, and other issues that can impact customers using those surfaces.

Everything else is out of scope. Examples of out-of-scope reports:

- Documentation, marketing site, blog, examples, or sample apps
- Issues that only affect a local or self-hosted setup you misconfigured
- Social engineering, physical attacks, or denial-of-service volume alone
- Findings in third-party dependencies with no demonstrated impact on the Unkey API or dashboard

If you are unsure whether something is in scope, email us anyway and say so. Prefer over-reporting to under-reporting for the API and dashboard.

## Supported Versions

Security fixes are applied to the current default branch (`main`) and to the latest released version. Older tagged releases are not maintained with backported security patches unless we say otherwise for a specific issue.

If you are self-hosting, run a current build of Unkey rather than an old tag.

## Reporting a Vulnerability

Please report security issues in the Unkey API or dashboard privately by emailing [security@unkey.com](mailto:security@unkey.com).

**Do not** open a public GitHub issue, discussion, or pull request for a vulnerability. This repository is not accepting external pull requests at this time, and public reports risk exposing users before a fix is available.

Include as much of the following as you can:

- A clear description of the issue and its impact on the API or dashboard
- Steps to reproduce (PoC, commands, or request samples)
- Affected component, version, commit, or deployment shape if known
- Any suggested fix or mitigation

## What to expect

- We aim to acknowledge reports within **48 hours**.
- We will keep you updated as we investigate and ship a fix.
- Please keep the issue private until we have released a fix and coordinated disclosure with you.

Thank you for helping keep Unkey and its users safe.
