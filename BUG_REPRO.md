# Bug
One reversal reason is normalized differently for the settlement, release entry, and audit event.

# Trigger
Reverse a confirmed settlement with a valid reason containing leading, trailing, repeated, and tab whitespace.

# Error
The three records store different reason strings for the same atomic operation.
