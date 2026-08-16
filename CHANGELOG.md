# Changelog

## v3.2 Public Beta

- Added **local Hevy workout CSV import** so Hevy Pro is no longer required.
- Added dual onboarding: Hevy Pro/API sync or free/local CSV import.
- CSV imports preserve workout title/date, exercise name, working-set index, load, reps, and RPE when exported.
- Warmup sets are excluded from working-set analysis, matching the API path.
- Pounds are converted to kilograms when the export provides pounds without kilograms.
- CSV mode infers current-program routine cards from the latest occurrence of recent workout names because Hevy workout exports do not contain saved routine definitions.
- CSV-derived routine cards and ChatGPT briefs are explicitly marked **inferred** rather than presented as direct routine syncs.
- CSV files are parsed locally and are not uploaded to a Hevy Signal cloud service.
- Added CSV-specific tests plus normal/race test coverage and runtime import validation.
- Rebuilt Windows x64, Windows ARM64, and macOS Apple Silicon packages.

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
