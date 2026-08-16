# Hevy Signal

Hevy Signal is a local training-analysis companion for Hevy. Hevy remains your workout logger; Signal analyzes training history, progression, volume, and exercise setup without generating a random replacement program.

> Public beta. Independent project; not affiliated with or endorsed by Hevy.

## Download

**[Download the latest Hevy Signal release](https://github.com/moriseren/hevy-signal/releases/latest)**

For v3.2, release assets should be:

- **Windows** — `Hevy-Signal-v3.2-Windows-all.zip` contains x64 (Intel/AMD) and ARM64 builds.
- **Mac Apple Silicon** — `Hevy-Signal-v3.2-macOS-Apple-Silicon.zip` for M1, M2, M3, M4, and newer Apple Silicon Macs.

You do **not** need Google Drive, Python, Node, Xcode, or an OpenAI API key. Builds are hosted directly by GitHub Releases.

Because the public beta is not code-signed yet, Windows SmartScreen or macOS Gatekeeper may show an unknown-developer warning. Only bypass that warning for a copy downloaded from this repository's official Releases page.

## Hevy Pro is optional in v3.2

Hevy Signal now supports two data-source modes.

### Hevy Pro — automatic API sync

Connect your own Hevy Developer API key. Signal syncs workout history plus your saved Hevy routines. In this mode, saved Hevy routines are the active program of record.

### Free Hevy — local workout CSV import

In Hevy, go to **Profile → Settings → Export & Import Data → Export Data → Export Workouts**. Then choose **Import Hevy CSV** inside Hevy Signal.

The CSV is parsed locally on your computer. It is not uploaded to a Hevy Signal server and no Hevy API key is required.

Hevy workout exports contain workout history but not saved-routine definitions. Because of that, CSV mode clearly labels its current-program cards as **inferred** from the latest occurrence of recent workout names. Review **Exercise setup** before relying on progression targets.

## What it does

- Generates deterministic progression signals from rep ranges, target sets, recent exposures, load × reps, and RPE when available.
- Supports multiple major muscle contributors plus supporting contributors for more realistic hypertrophy-volume accounting.
- Keeps distinct execution variants separate when Hevy exports them under distinct exercise names.
- Shows effective-set and stimulus summaries for movements in the current/inferred program.
- Lets you edit exercise mappings, stimulus type, programming role, fatigue cost, rep targets, progression set count, load jumps, and notes.
- Includes exercise-history views.
- Imports/exports exercise mappings without exporting the Hevy API key.
- Generates a local **ChatGPT Training Brief** you can copy or download as Markdown and bring into an existing ChatGPT conversation.
- Makes the ChatGPT brief source-aware: API mode identifies synced routines; CSV mode explicitly tells ChatGPT that program structure was inferred.
- Includes local privacy/reset controls.

## Quick start

### Windows

1. Open the [latest release](https://github.com/moriseren/hevy-signal/releases/latest) and download the Windows bundle.
2. Unzip it and use `x64` for almost all Intel/AMD Windows PCs, or `ARM64` for Windows-on-ARM / Snapdragon PCs.
3. Double-click `HevySignal.exe`.
4. Choose either **Sync Hevy** with a Hevy Pro API key or **Import Hevy CSV** without Pro.

### macOS

Open the [latest release](https://github.com/moriseren/hevy-signal/releases/latest), download the Apple Silicon ZIP, unzip it, and open `Hevy Signal.app`. Because the beta is unsigned, macOS may require Control-click → **Open** the first time.

Then choose either Hevy Pro API sync or local CSV import.

## ChatGPT handoff

Hevy Signal does not call the OpenAI API and does not require an OpenAI API key. After syncing/importing, open **ChatGPT handoff** and use:

- **Copy for ChatGPT** — copies a fresh structured training brief.
- **Download .md brief** — saves a Markdown brief you can upload to a ChatGPT conversation or Project.
- **Open ChatGPT** — opens ChatGPT in your browser.

The Hevy API key is never included in the training brief.

## Privacy

Hevy Signal runs locally. Your Hevy API key (if you use one), workout data, inferred/synced program state, and exercise mappings are stored in the app's local user-data directory. The app contacts Hevy only when you explicitly test or sync through the Pro/API path. CSV imports are parsed locally.

See [PRIVACY.md](PRIVACY.md) for more detail.

## Development

The app is written in Go with an embedded vanilla HTML/CSS/JavaScript frontend. A complete v3.2 source archive is provided with the release while the public repository source tree is being normalized.

## Release status

Current releases are unsigned public-beta builds with manual updates. Code signing/notarization is not yet configured, so Windows SmartScreen and macOS Gatekeeper warnings are expected.

## Disclaimer

Hevy Signal provides descriptive training analytics and progression signals. It is not medical advice and does not replace individualized professional coaching or healthcare guidance.
