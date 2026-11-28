### ci(sync): check upstream every twelve hours
- The upstream preview moves from a daily 09:23 UTC run to `0 3,15 * * *`,
  which is 00:00 and 12:00 at UTC-3 (03:00 and 15:00 UTC).
- The job itself is unchanged: ~11 s per run, guarded, opt-in publication and
  dedupe still apply. Twice a day still costs no billable minutes on a public
  repository.
