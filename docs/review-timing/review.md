# Review timing: UX adaptation

## Scope and reference

The two supplied WordUp screenshots show a recall count, a default next-review
action, and a selector for changing the interval. This review is limited to those
screenshots and the implemented Woordenboom flow. WordUp's actual scheduling
algorithm and interaction behavior were not tested.

The reference makes the default visible and gives learners control without
showing every option on the card. Its “Never, I won’t forget it” option makes a
strong claim about memory. Woordenboom uses reversible “Pause reviews” instead.
Its large centered dialog and colored illustrations are replaced with our black
study card and a white bottom sheet. No images or sound were added.

## Flow and findings

1. **Reveal the card.** The word keeps its existing position. The footer displays
   successful recalls, a recommended review interval, and a labeled Change
   control. Know saves the default in one tap; Not yet schedules ten minutes.
   Health: clear primary actions, quiet supporting text, no extra confirmation
   for the normal path. See `01-card.png`.
2. **Choose an interval.** Change opens a white sheet with explicit radio choices,
   a Recommended label, Cancel, and Remember & continue. Choices range from ten
   minutes to a year. The list scrolls on short screens while confirmation stays
   available. Cancel does not record an answer. Health: clear selection and
   reversible dismissal. See `02-options.png` and `04-small-screen.png`.
3. **Pause or resume.** Pause reviews explains that the word remains in the map.
   Remember & pause records the recall and excludes the word from automatic
   sessions. Choosing a timed interval through the map resumes reviews. Health:
   the consequence is explained before saving. See `03-pause.png`.
4. **Save and continue.** Confirmation saves the answer and timing together before
   advancing. Failed saves keep the word and selected timing available for retry.
   No due words produces a “You’re up to date” screen, rather than early repeats.
   Health: errors remain recoverable and the chosen timing is respected.
   See `05-up-to-date.png` and `checks.json`.

## Scheduling policy

The default ladder is 1, 3, 7, 14, 30, 60, 90, 180, then 365 days, based on
consecutive successful recalls. It remains at 365 days after reaching that limit. A failed recall resets that streak and schedules ten minutes. The visible
remembered count is the total successful recalls, not the current streak. Manual
intervals are per answer and do not accelerate the recall streak. Future dates and
paused words are excluded even when there are no unseen words left.

Day intervals are elapsed durations; one day means 24 hours. Month choices use
30, 60, 90, and 180 days; a year is 365 days. Existing due dates are preserved.
The API still accepts the legacy two-day option for older clients. This is a transparent
product policy, not a claim that we have fitted an individual memory model.
A selected schedule controls future in-app practice. It does not send reminders
or notifications.

## Accessibility and verification limits

Controls have explicit button/radio roles, labels, and selected states. Radio rows
are at least 48 points high. The sheet uses React Native Modal, supports cancel and
platform back dismissal, and keeps a scrolling list on short screens. Text and
selection use the existing monochrome tokens; color is not the only state cue.

Automated checks cover backend timestamps, recommendation growth, streak reset,
pausing, the one-year cap, invalid input, duplicate session answers, and session selection. Live
browser checks cover manual week, three-month, and one-year scheduling, cancel, pause/resume, failed-save retry,
small screens, and Escape dismissal. The existing mobile regression suite checks
320/375/414/768 widths, stable word positioning, reduced motion, and stack flows.
Native bundles are exported separately. These checks do not establish VoiceOver
or TalkBack compliance; native assistive-technology testing remains unverified.
