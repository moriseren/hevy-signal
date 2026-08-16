# Privacy

Hevy Signal is designed to run locally on the user's computer.

## Data stored locally

Depending on which mode the user chooses, Hevy Signal stores the following in the user's local application-data directory:

- Hevy API key, only if the user chooses Hevy Pro/API sync
- synced or CSV-imported workout history
- synced routine state or CSV-inferred current-program state
- exercise classifications and custom mappings
- local app state needed for analysis

On Windows this is normally stored under the user's local configuration directory in a `Hevy Signal` folder. On macOS the current beta stores data under `~/.hevy-signal`.

## Network access

### Hevy Pro/API mode

Hevy Signal contacts the Hevy API when the user explicitly tests their Hevy connection or runs a sync.

### CSV mode

A Hevy workout CSV is read and parsed by the local Hevy Signal process. The CSV is not uploaded to a Hevy Signal cloud service and no Hevy API key is required.

Hevy Signal does not require or use an OpenAI API key.

The **ChatGPT handoff** is generated locally as plain text/Markdown. The user decides whether to copy or upload that brief to ChatGPT.

## Exports

Exercise-mapping exports are intended to contain exercise metadata only. They do not include the saved Hevy API key.

The ChatGPT training brief also excludes the Hevy API key.

## Local deletion

The app includes controls to delete training data while keeping a saved Hevy connection, or to fully reset local app data including the saved Hevy API key.

## Security note

Current public-beta builds are unsigned and distributed with manual updates. Do not enter a Hevy API key into a copy obtained from an untrusted source. Verify downloads against published checksums when available.

## Independence

Hevy Signal is an independent project and is not affiliated with or endorsed by Hevy.
