# Auth gate drive

Feature: features/auth-gate.md
Verdict before drive: auth-landing-only (see launch.txt). Canonical preflight blocked because docker is not on PATH (see doctor-preflight.txt).

Action: opened http://127.0.0.1:3000/auth/sign-in in headless Chrome and read the accessibility tree. Did not activate Continue to dashboard.

Resulting state:
- heading role=heading name=Local dashboard
- link role=link name=Continue to dashboard href=/apis
- description: Authentication is disabled in local mode. Continue with the built-in local account.

Screenshot: sign-in.png
Saved document: sign-in.html
Machine-readable notes: drive-auth-gate.json
