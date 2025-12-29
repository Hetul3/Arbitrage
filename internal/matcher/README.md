# internal/matcher

Cross-venue similarity helpers that sit on top of the Chroma client. The current
implementation queries the vector store right after a snapshot is embedded and
upserted so we can surface potential Polymarket ↔ Kalshi matches in real time.

- `Finder` – small helper that queries Chroma with the snapshot’s embedding,
  filters for opposite-venue markets within a freshness window, and returns the
  closest result that clears the configured similarity threshold.
- `Logger` – emits match summaries (or full payload dumps) based on the current
  log mode so prod/dev/verbose commands can share the same plumbing.

TODO: extend this package to check Redis for cached SAFE/UNSAFE verdicts before
calling the LLM, reuse cached bundles when hashes match, and record new verdicts
after the arb + LLM stages.
