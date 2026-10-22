# Security Policy

## Reporting a Vulnerability

Do **not** open a public issue for genuine security vulnerabilities. Instead, report them privately via [GitHub Private Vulnerability Reporting](https://github.com/spf13/afero/security/advisories/new).

## Scope & Threat Model

Afero is a Go filesystem abstraction library, not a sandboxed runtime or network service. Callers are responsible for validating untrusted input before passing it to Afero APIs unless explicitly using a confining wrapper such as `BasePathFs` or `ReadOnlyFs`.

### In Scope (Report Privately)

- Escapes from explicit security/confinement boundaries (e.g., `BasePathFs` path traversal or `ReadOnlyFs` write bypass).
- Unintended privilege escalation or credential exposure caused by Afero internals when used as documented.

### Out of Scope (Open a Regular GitHub Issue)

The following are treated as ordinary robustness bugs, **not** CVE-eligible security vulnerabilities:

- Path traversal when callers pass unvalidated user input directly to unconfined filesystems (`OsFs`, `MemMapFs`, `TempFile`/`TempDir` patterns, etc.).
- Panics, `EOF`, or memory exhaustion when feeding untrusted or corrupted archives into `tarfs` or `zipfs`.
- Automated scanner findings without a realistic, exploitable trust-boundary violation.
