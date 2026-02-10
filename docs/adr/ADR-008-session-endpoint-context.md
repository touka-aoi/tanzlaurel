# ADR: SessionEndpoint の Context 運用

# Status
- Draft

# Decision
Context の扱いについて以下の点を明記する。
1. 各コネクションのコンテキストがキャンセルされるのは、接続が閉じられた時、またはサーバーが閉じられた時のみとする。
2. Contextを閉じる際は必ず、接続のリソースをクリーンアップしなければならない。

# Context

# Consideration

# Consequences

# References

