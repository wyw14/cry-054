# Bug
A review notification escapes even when the surrounding transaction rolls back.

# Trigger
Complete a valid review decision while forcing the final audit append to fail.

# Error
The claim returns to its prior state, but the notifier contains a false review-completed message.
