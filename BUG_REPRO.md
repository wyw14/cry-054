# Bug
Cancellation drops due scheduler jobs that were detached but never started.

# Trigger
Schedule two jobs for the same instant and let the first completed operation cancel the run context.

# Error
The second run cannot find the unstarted job, so only the first operation ever executes.
