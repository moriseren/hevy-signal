# Privacy

Hevy Signal is designed to run locally on the user's computer.

## Data stored locally

Hevy Signal stores the following in the user's local application-data directory:

- Hevy API key
- synced workout history
- synced current routines
- exercise classifications and custom mappings
- local app state needed for analysis

On Windows this is normally stored under the user's local configuration directory in a `Hevy Signal` folder. On macOS the current beta stores data under `~/.hevy-signal`.

## Network access

Hevy Signal contacts the Hevy API when the user tests their Hevy connection or runs a sync. The app does not require or use an OpenAI API key.

The **ChatGPT handoff** is generated locally as plain text/Markdown. The user decides whether to copy or upload that brief to ChatGPT.

## Exports

Exercise-mapping exports are intended to contain exercise metadata only. They do not include the saved Hevy API key.

## Local deletion

The app includes controls to delete synced training data while keeping the Hevy connection, or to fully reset local app data including the saved Hevy API key.

## Security note

Current public-beta builds are unsigned and distributed with manual updates. Do not enter a Hevy API key into a copy obtained from an untrusted source. Verify downloads against published checksums when available.

## Independence

Hevy Signal is an independent project and is not affiliated with or endorsed by Hevy.
