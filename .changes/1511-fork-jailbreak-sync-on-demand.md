### feat(jailbreak): the preset library syncs on demand, and refreshes only while a combo uses it

- **Sync now** now downloads every source, including a paused one, so the library can be fetched the first time, before any combo can use a preset from it.
- The automatic refresh (on boot and every hour) runs only for an enabled source that at least one combo is bound to through a preset it wrote. A source nobody uses is not fetched on a schedule, so a library that is no longer in use stops calling the outside.
