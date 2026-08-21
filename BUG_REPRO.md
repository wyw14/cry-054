# Bug
Concurrent confirmations share mutable request-progress state inside one service instance.

# Trigger
Release two confirmation goroutines for independent claimants through the same start barrier under the race detector.

# Error
The race detector reports concurrent writes in confirmation progress tracking.
