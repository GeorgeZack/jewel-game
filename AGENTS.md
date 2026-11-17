# Project instructions

## GUI debugging

Launch GUI applications and test windows on Hyprland workspace 9
(the debug workspace), without switching the active workspace.

Do not launch debug windows into the current workspace: they resize
my working windows. Use Hyprland launch rules to place them on
workspace 9 silently from the moment they open.

## Testing a branch in a worktree

I test with `./start.sh` in the main checkout, always the same command. The file
`.start-worktree` there (untracked) names the worktree that the script runs in place of the
main checkout: a folder name under `.claude/worktrees/`, or a path.

- When work on a branch in a worktree is ready for me to try, write that worktree's folder
  name into `.start-worktree` and tell me which branch `./start.sh` now runs.
- When the branch is merged or its worktree is removed, delete `.start-worktree`, so the
  script runs `master` again.
- `./start.sh --master` runs the main checkout whatever the file says.

The game uses my real save in `~/.config` either way.
