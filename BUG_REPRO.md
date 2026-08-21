# Bug
A failed atomic attachment promotion leaks its temporary upload file.

# Trigger
Block the intended final path with a directory, then save a valid attachment whose rename must fail.

# Error
The save returns a promotion error but `.upload-*` remains under the configured storage root.
