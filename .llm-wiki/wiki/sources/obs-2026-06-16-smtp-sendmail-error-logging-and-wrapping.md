---
type: source
title: "Observation: SMTP SendMail error logging and wrapping"
slug: obs-2026-06-16-smtp-sendmail-error-logging-and-wrapping
status: observation
created: 2026-06-16
updated: 2026-06-16
relevance: medium
observed_at: 2026-06-16T16:24:29.473Z
tags: ["backend", "smtp", "observability"]
source_context: "Fixing SMTP error observability per user request"
---
# 🔍 Observation: SMTP SendMail error logging and wrapping
Added structured slog.Error logging with smtp host, port, from, to, subject, and wrapped the error with fmt.Errorf("smtp send: %w") in SMTPEngine.Send() at server/internal/engine/smtp.go. Previously the raw smtp.SendMail error was returned with no logging or wrapping. Both callers (worker.go, module_auth.go) now get richer errors upstream.
*Relevance: medium*

*Context: Fixing SMTP error observability per user request*

*Tags: backend smtp observability*
---
*Observed: 2026-06-16T16:24:29.473Z*