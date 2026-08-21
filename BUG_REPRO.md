# Bug
Canceled controlled exports still persist a CSV file and append a successful audit event.

# Trigger
Cancel the request context before starting an authorized settlement export.

# Error
The export returns success and leaves both file and audit side effects instead of returning `context canceled`.
