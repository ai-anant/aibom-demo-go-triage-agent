# Customer Tier Commitments

Enterprise customers have a one-hour first-response target for Sev-1 tickets and
a four-hour target for Sev-2. Business customers are four hours and one business
day respectively. Starter customers are best-effort.

Enterprise tickets always receive a named owner. Escalations from Enterprise
customers must be acknowledged in the ticket before any customer-visible reply.

A ticket from an Enterprise customer marked urgent is never auto-closed; a human
reviews it even when the model confidence is high.

Security incidents raised by any tier bypass the standard queue and go straight
to the security on-call rota.
