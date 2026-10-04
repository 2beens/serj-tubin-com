# Gymstats improvements

Review of the Vue log/list/stats UI (`serj-tubin-vue/src/components/gymstats`) against the Go API (`internal/gymstats`). Source is the current tree, not production rows.

Backend and Vue items below are marked done. Still open: public exercise photos (skipped on purpose) and storing fractional kilos (the column is still `INTEGER`).

## Bugs

### Log stays on the skeleton after a refresh — frontend done

`ExercisesLog.vue` — `getExercises` runs in the child `mounted` hook and returns immediately when `$root.loggedIn` is false, after already setting `loadingData`. `App.vue` only sets `loggedIn` from the session cookie in its own `mounted` hook, which runs later. A full reload of `/gymstats` on the Log tab never retries, so the skeleton stays up. Opening Stats and coming back works, because by then the cookie check has finished.

### The last page of the log repeats earlier sets — backend done

`exercises/exercises_repo.go` `List` — when the remaining rows are fewer than the page size, `offset` is pulled backward so the page is always full. With 120 sets and a page size of 50, page 3 starts at row 70 and overlaps page 2. The integration test expects this (`test/gymstats_exercises_test.go` comments that page 2 of 15 items moves the offset from 10 to 5). The Vue pager does not, so the end of the log shows sets you already saw.

### "Today" is a UTC day, and the rest hack adds time — backend done

`exercises/exercises_handler.go` `HandleAdd` — `todayMidnight` is `time.Now()` in Berlin, then `Truncate(24h)`. `Truncate` cuts to UTC midnight, so in summer the window is 02:00–02:00 Berlin, not midnight. A normal evening session still lands inside it. A set between midnight and 02:00 is grouped with the previous day. The same `Truncate` buckets history and the rest chart (`exercises/analyzer.go`). The +1h/+2h patch then runs on any negative gap in that range, so a real negative gap gets turned into a positive rest.

### The previous-set lookup ignores the today window — backend done

`exercises/exercises_repo.go` `List` — `HandleAdd` asks `List` for today's two newest sets. `ExercisesCount` applies `From` and `To`. The `SELECT` in `List` does not; those parameters are never bound. The rest timer therefore uses the two newest rows in the table. The first set of the day still shows "no previous set", because a count of 1 shrinks the limit to 1. Later sets are right only while those two newest rows really are today's.

### Update reports 500 for a missing set and ignores other read errors — backend done

`exercises/exercises_handler.go` `HandleUpdate` — the condition is `err != nil && errors.Is(not found)`. `HandleDelete` uses `err != nil && !errors.Is`. A missing id returns 500, and the 404 branch under it never runs. Any other `Get` error falls through and `Update` still runs.

### The table pencil does nothing — frontend done

`serj-tubin-vue/src/components/gymstats/ExercisesLog.vue` — `editExercise` only `console.warn`s, with a TODO. Long-press and right-click on the list view open `EditExercise`. The pencil in the desktop table does not. Inline cell editors still save.

### Renaming a type says it saved and writes nothing — done

`ExerciseSetup.vue` and `exercises/exercise_types_repo.go` `UpdateExerciseType` — the update sets `exercise_id` and `muscle_group` to the same values it filters on, and the handler ignores rows affected, so a changed id returns 204 and leaves the row. The muscle-group select is `return-object`. After you pick a group, the PUT body sends an object. The API field is a string, so that request is a 400. Delete then puts that object into the URL.

### Exercise photos are public (skip this)

`internal/middleware/auth.go` — auth skips `^/gymstats/image/\d+`. Upload stores the file with `IsPrivate: false`. Anyone who can guess or see the id can fetch the image without `X-SERJ-TOKEN`.

### A non-string metadata value crashes the scan — backend done

`exercises/exercises_repo.go` `rows2exercises` — each JSON value is asserted with `v.(string)`. One number or boolean in metadata fails the whole list or get, not just that row. The Vue app writes strings, so this bites old or hand-edited rows.

### The event insert is outside its transaction — backend done

`events/repo.go` `Add` — `Begin` is on the pool, then `Query` uses `r.db`, not the tx. A later failure cannot roll the insert back. The rollback branch checks `err != nil`, which is already true, so a successful rollback is reported as a rollback failure. The Vue log does not call this; it is the iOS training/weight/pain path.

## Improvements

### The testing filter is an OR of nullable comparisons — backend done

`exercises/exercises_repo.go` and `events/repo.go` — `exclude_testing_data` keeps a row when `testing != 'true' OR test != 'true'`. A missing key is NULL, and NULL fails the WHERE, so a row with neither key disappears from stats. A row stays if only one flag is the string `false`, even when the other is `true`. Vue sends `{"env":"prod","testing":"false"}`, which passes. `COALESCE` and `AND` would match the comment on the flag.

### The rest chart treats long breaks as set rest — backend done

`exercises/analyzer.go` `AvgSetDuration` — every gap between consecutive sets on a UTC day is averaged, including the pause between exercises or a morning and evening session. Days with one set are skipped. The headline number is the mean of those daily means, so a two-set day weighs the same as a twenty-set day. Avg kilos and reps are integer division, so 12 and 13 become 12.

### Two exercise catalogs — frontend done

`serj-tubin-vue/src/gymstats/index.js` — Add falls back to the hardcoded list when `/gymstats/types` fails, and `EditExercise` reads `localStorage` or that same list, not the API. A type you added on another device is missing from the editor until Refresh has written `localStorage` on this browser.

### Weight is a whole number, and negatives are allowed — negatives rejected; fractional kilos still open

`AddExercise.vue` and `Exercise.Kilos` — kilos and reps are integers. 1.25 kg plates cannot be stored. The add button treats any non-zero number as valid, including negatives. The API does not check either.

### A failed type refresh still says it worked — frontend done

`AddExercise.vue` `refreshExerciseTypes` — the catch sets an error snackbar, then `finally` overwrites it with "Exercise types refreshed!". `saveExercise` also `JSON.parse`s metadata with no try/catch, so a bad edit throws and shows nothing. `secondsSincePreviousSet || -1` treats a 0-second gap as "no previous set".

### Image upload and delete can leave orphans — backend done

`exercises/exercise_types_handler.go` — upload saves the file, then inserts the row. A failed insert leaves the file. Delete joins errors with `errors.Join(err)`, which replaces the accumulator instead of appending, then aborts before the type row is removed. If the images folder is missing at upload time, `imagesFolder` is nil and `Save` panics. Startup creates that folder, so this is only a drifted disk.
