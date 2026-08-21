# Bug
The API timeout boundary replaces the public ingress request identity.

# Trigger
Send a canceled API request with a fixed `X-Request-ID` through the normal router.

# Error
The response header and JSON error body contain a derived `-timeout` identity instead of the ingress value.
