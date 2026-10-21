HEVY SIGNAL v3.2 PUBLIC BETA

Hevy Signal is a local training-analysis companion for Hevy.

YOU NOW HAVE TWO WAYS TO USE IT:

1) Hevy Pro / automatic sync
   - In Hevy: Settings -> Developer -> copy your API key.
   - In Hevy Signal: Settings -> paste your own key -> Save & test Hevy -> Sync Hevy.
   - Saved Hevy routines define the current program.

2) No Hevy Pro / local CSV import
   - In Hevy: Profile -> Settings -> Export & Import Data -> Export Data -> Export Workouts.
   - In Hevy Signal: click Import Hevy CSV and choose that file.
   - The CSV is parsed locally on your computer. It is not uploaded to a Hevy Signal server.
   - Workout exports do not include saved routine definitions, so Signal clearly marks a current program inferred from recent workout history. Review Exercise setup before relying on progression targets.

CHATGPT HANDOFF
- No OpenAI API key is needed.
- Signal creates a local Markdown training brief that you can copy/download and bring into your existing ChatGPT chat.
- The Hevy API key is never included in that brief.

PRIVACY
- Training state is stored locally in the app's user-data folder.
- The app contacts Hevy only for API test/sync when you choose the Pro path.
- CSV imports are local.
- Hevy Signal does not send training data to an independent Hevy Signal cloud service.

PUBLIC BETA NOTES
- Builds are unsigned. Windows SmartScreen/macOS Gatekeeper can warn about an unknown publisher/developer.
- Only bypass those warnings for a copy downloaded from the official moriseren/hevy-signal GitHub Releases page.
- Independent project; not affiliated with or endorsed by Hevy.
