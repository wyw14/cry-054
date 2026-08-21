# Bug
Duplicate receipt submissions lose the duplicate-claim conflict identity and return a generic validation response.

# Trigger
Submit one valid claim, then submit the same receipt digest for the same claimant and project with a different idempotency key.

# Error
The second request returns HTTP 500 with `VALIDATION_FAILED` instead of HTTP 409 with `DUPLICATE_CLAIM`.
