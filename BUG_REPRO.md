# Bug
Open-ended published rules panic during preview applicability evaluation.

# Trigger
Calculate a preview for an in-scope claim using a rule whose effective end date is absent.

# Error
The process raises `runtime error: invalid memory address or nil pointer dereference`.
