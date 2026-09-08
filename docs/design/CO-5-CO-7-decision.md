# Design Note: CO-5 and CO-7 Decisions

Date: 2026-08-21

Decision summary
----------------

- CO-5: The positional-path-plus-subcommand model is retained. The team
  decided not to introduce `--location` or `--mode` flags. This decision was
  recorded and agreed on 21 August 2026 (D7).

- CO-7: The `DEVELOPMENT_STATUS.md` file was found to be out of date and
  potentially misleading. It has been archived and rewritten as an archived
  snapshot in the repository root (see `DEVELOPMENT_STATUS.md`). If desired,
  the file may be deleted in a follow-up commit.

Rationale
---------

CO-5 rationale:

- The positional-path-plus-subcommand usage keeps CLI ergonomics simple for
  scripted and interactive workflows.
- Adding `--location` / `--mode` introduced ambiguity and increased the
  surface-area for flags without delivering clear UX benefits.
- The retained model preserves backward compatibility and avoids adding
  additional parsing/validation complexity.

CO-7 rationale:

- The original `DEVELOPMENT_STATUS.md` (2026-05-02) listed items as "Next Up"
  that have since shipped, which risks confusing downstream readers and the
  client.
- Archiving the file preserves historical context while directing readers to
  the issue tracker and `CHANGELOG.md` for the current roadmap.

Follow-up actions for maintainers
--------------------------------

1. Close the GitHub issues associated with CO-5 and CO-7, and in each issue
   comment link to this design note. Suggested comment text:

   "Decision: retained positional-path-plus-subcommand model; not adding
   `--location`/`--mode`. See docs/design/CO-5-CO-7-decision.md (2026-08-21)."

2. If you want the repository to remove `DEVELOPMENT_STATUS.md` entirely,
   open a follow-up PR that deletes the file and references this design note.

3. If any stakeholder disagrees with the CO-5 decision, open an RFC issue
   referencing this note and describe the proposed alternative with examples.

Authors
-------

Record created by maintainers on 2026-08-21.
