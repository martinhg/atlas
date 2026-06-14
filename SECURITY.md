# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |

## Scope

The following components are covered by this security policy:

- **Atlas Server** — API server (`cmd/atlas-server/`)
- **Atlas CLI** — command-line tool (`cmd/atlas/`)
- **Atlas Web** — frontend application (`web/`)
- **Migrations** — database schema (`migrations/`)

Third-party dependencies are covered when the vulnerability is exploitable through Atlas.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Instead, please report them responsibly by emailing: **security@nesbite.com**

Include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your report within 48 hours and provide a detailed response within 5 business days.

## Disclosure Policy

- We follow responsible disclosure practices.
- We will credit reporters in security advisories (unless anonymity is requested).
- We aim to release fixes within 7 days of confirming a vulnerability.
