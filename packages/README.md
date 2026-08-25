# Local docreader packages

This directory is intentionally kept in the build context because the
docreader image copies optional offline protoc archives from `packages/`.
Place `protoc-3.19.4-linux-<arch>.zip` here for an offline build; an empty
directory falls back to the documented network download during image build.
