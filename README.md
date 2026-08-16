# Hevy Signal

Hevy Signal is a local training-analysis companion for Hevy. Hevy remains your workout logger; Signal treats your **current saved routines as the program of record** and uses workout history as performance evidence.

> Public beta. Independent project; not affiliated with or endorsed by Hevy.

## Download

**[Download the latest Hevy Signal release](https://github.com/moriseren/hevy-signal/releases/latest)**

On the release page, choose the file for your computer:

- **Windows x64** — for almost all Intel/AMD Windows PCs.
- **Windows ARM64** — for Windows-on-ARM / Snapdragon PCs.
- **Mac Apple Silicon** — for M1, M2, M3, M4, and newer Apple Silicon Macs.

You do **not** need Google Drive, Python, Node, Xcode, or an OpenAI API key. The downloadable builds are hosted directly by GitHub Releases.

Because the public beta is not code-signed yet, Windows SmartScreen or macOS Gatekeeper may show an unknown-developer warning. Only bypass that warning when the app was downloaded from this repository's official Releases page.

## What it does

- Focuses analysis on exercises that are actually in your current Hevy routines.
- Generates deterministic progression signals from rep ranges, target sets, recent exposures, load × reps, and RPE when available.
- Supports multiple **major muscle contributors** plus supporting contributors for more realistic hypertrophy-volume accounting.
- Keeps execution variants separate, such as lean-forward vs reclined hip abduction.
- Shows effective-set and stimulus summaries for movements in the current program.
- Lets you edit exercise mappings, stimulus type, programming role, fatigue cost, rep targets, progression set count, load jumps, and notes.
- Includes an exercise-history viewer.
- Imports/exports exercise mappings without exporting the Hevy API key.
- Generates a local **ChatGPT Training Brief** you can copy or download as Markdown and bring into an existing ChatGPT conversation. No OpenAI API key is required.
- Includes local privacy/reset controls.

## Requirements

A Hevy Pro account with access to Hevy's developer API key is required for syncing.

## Quick start

### Windows

1. Open the [latest release](https://github.com/moriseren/hevy-signal/releases/latest) and download the Windows x64 ZIP for a normal Intel/AMD Windows PC, or ARM64 for Windows-on-ARM.
2. Unzip it to a normal folder.
3. Double-click `HevySignal.exe`.
4. Windows SmartScreen may warn because the public beta is not code-signed yet. Only choose **More info → Run anyway** if you downloaded it from this repository's release page.
5. In Hevy, open **Settings → Developer** and copy your API key.
6. In Hevy Signal, open **Settings**, paste the key, then **Save & test Hevy**.
7. Click **Sync Hevy**.
8. Check **Exercise setup** for movements that could not be classified confidently.

### macOS

Open the [latest release](https://github.com/moriseren/hevy-signal/releases/latest), download the Apple Silicon ZIP, unzip it, and open `Hevy Signal.app`. Because the beta is unsigned, macOS may require Control-click → **Open** the first time.

## ChatGPT handoff

Hevy Signal does not call the OpenAI API and does not require an OpenAI API key. After syncing, open **ChatGPT handoff** and use one of these options:

- **Copy for ChatGPT** — copies a fresh structured training brief.
- **Download .md brief** — saves a Markdown brief you can upload to a ChatGPT conversation or Project.
- **Open ChatGPT** — opens ChatGPT in your browser.

The brief distinguishes programmed routine structure from completed training history and contains no Hevy API key.

## Privacy

Hevy Signal runs locally. Your Hevy API key, synced workout/routine data, and exercise mappings are stored in your local user data directory. The app contacts Hevy only when you test the connection or sync. Mapping exports do not contain your Hevy API key.

See [PRIVACY.md](PRIVACY.md) for more detail.

## Development

The app is written in Go with an embedded vanilla HTML/CSS/JavaScript frontend.

Run tests with:

```bash
go test ./...
```

The public beta is intentionally simple: no external database, no OpenAI SDK, and no package-manager runtime is required for end users.

## Release status

Current releases are unsigned public-beta builds with manual updates. Code signing/notarization is not yet configured, so Windows SmartScreen and macOS Gatekeeper warnings are expected.

## Disclaimer

Hevy Signal provides descriptive training analytics and progression signals. It is not medical advice and does not replace individualized professional coaching or healthcare guidance.
