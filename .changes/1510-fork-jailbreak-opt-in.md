### feat(jailbreak): off by default, chosen per combo

- Preset keyword rules no longer arm requests unless a new global switch is on (`/api/jailbreak/keywords`, off by default), and the preset library sync is seeded disabled, so a fresh install no longer downloads third-party prompts and injects them into models nobody selected. Exact bindings are unaffected.
- The combo edit dialog gets a **Prompt injection** selector (Off, Default document, or a preset), stored as a binding on the combo name, which arms every leg; a renamed combo keeps it. The Persona page gets the keyword switch, and the log names the preset applied.
- Existing installs keep their presets and bindings; keyword matching simply turns off until it is switched on.
