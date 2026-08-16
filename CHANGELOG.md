# Changelog

## v3.1 Public Beta

- Added polished Windows x64 and Windows ARM64 builds.
- Kept the v3 local-analysis architecture with no OpenAI API dependency.
- Added Windows-native local data storage behavior.
- Added duplicate-launch reuse on Windows.
- Hardened repeated JSON state-file writes on Windows.
- Retained routine-first analysis, execution variants, major/supporting muscle contributors, effective-set volume accounting, progression signals, mapping import/export, reset controls, and ChatGPT handoff.

## v3.0

- Removed Coach Chat and OpenAI API-key handling entirely.
- Replaced in-app AI chat with local ChatGPT handoff: copy brief, download Markdown brief, and open ChatGPT.
- Removed persistent background-service architecture from the Mac build.
- Added local quit control.
- Added migration that removes legacy OpenAI configuration fields while preserving the Hevy key.

## v2.x

- Switched to current-routine-first analysis.
- Added routine syncing from Hevy.
- Added multiple major muscle contributors plus supporting contributors.
- Added exercise mapping editor and import/export.
- Added exercise history and improved progression logic.
- Fixed empty/unweighted-history crashes and Hevy pagination edge cases.
- Added Mac and Windows public-beta packaging.

## v1.x

- Introduced deterministic progression signals.
- Added movement/stimulus metadata and Wellness-oriented exercise mappings.
- Added execution-variant handling and effective-volume concepts.

## v0.x

- Initial local Hevy sync and training-analysis prototype.
