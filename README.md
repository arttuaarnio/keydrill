# Keydrill

Practise shell and Vim commands in your terminal until they stick.

Keydrill is a small terminal game. Shell drills hand you a throwaway folder of
files and check the result of the real commands you run there, so any correct
approach counts. Vim drills come later: they will run on a built-in simulator,
so you type real Vim keys and score by how few keystrokes you need, with no Vim
install required.

> **Status:** early development. The first shell topic, Moving around, is
> playable; more topics are being added one at a time.

## Install

Requires Go 1.25 or newer, and macOS or Linux for the shell drills.

```bash
go install github.com/arttuaarnio/keydrill@latest
keydrill
```

Or run it from a clone:

```bash
git clone https://github.com/arttuaarnio/keydrill
cd keydrill
go run .
```

## Shell drills

The shell course is a run of topics, easiest first. Each topic starts with a
briefing on its commands, followed by drills that get harder, ending in an
exam. Solve a drill to unlock the next; pass the exam to unlock the next topic.

Every command you run counts. Solve a drill at par for ★★★, within two
commands of par for ★★, and anything more for ★. Using the hint caps you at ★★.

Commands run in your own `$SHELL` (sh, bash, zsh, dash or ksh; anything else
falls back to `/bin/sh`), from a temporary folder that acts as your home
folder for the drill.

### Safety

Each drill gets its own folder in the system temp directory, which is deleted
when you leave the drill. Keydrill only ever deletes folders it created there.
`~` and `HOME` point at that folder, a command that leaves it puts you back at
`~`, and each command is stopped after 5 seconds, with file size capped and
anything it left running in the background ended.

It's a sandbox folder, not a locked-down jail: your commands are real, so an
absolute path outside it, like `rm /Users/you/file`, would still work. Stick to
the drill folder.

## Controls

| Key | Action |
| --- | --- |
| `j` / `k` or arrow keys | Move |
| `enter` | Select |
| `esc` | Back |
| `q` | Quit |

Inside a drill you type commands as in a normal shell. `↑` and `↓` step
through earlier commands and `ctrl+c` clears the line. These words are game
controls and don't count as commands:

| Type | To |
| --- | --- |
| `help` | Show the topic's briefing |
| `hint` | Show a solution (caps you at ★★) |
| `reset` | Start the drill over |
| `clear` | Clear the screen |
| `exit` | Leave the drill |

## Progress

Your best result for each drill is saved in `progress.json` in your user
config folder: `~/Library/Application Support/keydrill/` on macOS,
`~/.config/keydrill/` on Linux. Delete it to start over.

## Roadmap

- [ ] Shell course
  - [x] Moving around: `pwd` `ls` `cd`
  - [ ] Making things, moving, copying, deleting, wildcards
  - [ ] Reading and redirecting, `grep`, `find`, pipes, `xargs`
- [ ] Stats: skill level and weak spots
- [ ] Vim motion drills
- [ ] Vim editing drills: operators, text objects, insert mode

## Built with

[Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss) by Charm.

## Licence

[MIT](LICENSE)
