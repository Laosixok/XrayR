# Repository boundaries

- Keep changes within the user's requested scope. The neighboring panel checkout is read-only unless the user explicitly requests panel changes.
- Public-facing panel names must use SSPanel-Metron. Local directory names are not product names.
- Never put private domains, real node addresses, credentials, API keys, certificate private keys, personal UUIDs or production configuration into source, examples, tests, documentation, commit messages, release notes or uploaded artifacts.
- Use reserved example.com / example.net / example.org domains and fictional identifiers. Use 80 for plain WebSocket examples and 443 for TLS WebSocket examples.
- Before every commit or upload, review the staged diff and the exact artifact contents for scope and privacy. Do not stage unrelated files, private configurations, tool caches or local artifacts.
- If private data has entered Git history, a new deletion commit is insufficient. Rewrite affected reachable history and audit branches and tags; explain any hosting-side cache limitation accurately.
- Preserve existing panel APIs and legacy backend compatibility unless the user explicitly approves a breaking change. Do not change unrelated panel files or deployment settings.
- Preserve upstream and dependency license notices. Keep release version declarations and provenance accurate.
