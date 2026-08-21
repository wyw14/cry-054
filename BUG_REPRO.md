# Bug
Published rule segment arrays escape through repository read results.

# Trigger
Modify a segment in one result from listing rules, then preview another claim using the same published rule.

# Error
The later preview uses the caller's modified rate and produces an incorrect approved amount.
