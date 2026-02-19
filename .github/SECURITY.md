# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in pollhook, please report it responsibly.

**Email:** security@continue.dev

We will acknowledge your report within 48 hours and provide a timeline for a fix.

## Scope

Security issues we care about:

- **Command injection** via config file values or environment variable expansion
- **Secret leakage** in logs, state files, or error messages (webhook secrets, API tokens)
- **State file tampering** leading to replay attacks or missed events
- **SSRF** via crafted webhook URLs or command output

## Out of Scope

- Vulnerabilities in upstream dependencies (report to the upstream project)
- Issues requiring local access to the machine running pollhook
- Denial of service via large API responses (bounded by command timeout)
