# Plugin distribution — the skill without the binary

Status: **live**. Self-hosted marketplace works today; community-marketplace
submission is a form, not a pull request.

## 1. Why — reach beyond `rekal init`

Today the skill reaches a repository exactly one way: the binary installs it.
`rekal init` writes `.claude/skills/rekal/`, and a version-pinned marker lets
recall refresh it in place after an upgrade. That path is correct and stays the
default — but it only exists for someone who already has the binary.

A Claude Code plugin inverts the order. The agent gets the skill first, from a
marketplace, and the binary comes after. That is the discovery path: someone who
has never heard of Rekal installs a plugin, and the skill tells their agent what
Rekal is and how to set it up.

Nothing about the store, the wire format, or capture changes. This is packaging.

## 2. Shape — the plugin bootstraps, the binary owns the skill

```
plugin/                               ← plugin root: setup only
├── .claude-plugin/plugin.json
├── SKILL.md                          ← one skill, name: rekal-setup
└── README.md

.claude-plugin/marketplace.json       ← repo root: the catalog listing it
cmd/rekal/cli/skill/skills/rekal/     ← the recall skill — binary only, not shipped here
```

The plugin carries no `skills/`, no `references/`, no `scripts/`. It explains
what Rekal is, installs the binary, runs `rekal init`, and hands off. From that
point the binary owns the recall skill, exactly as it always has.

**Why not ship the recall skill in the plugin.** The first design did, with the
skill dir doubling as the plugin root — one directory, two consumers, no copy to
drift. It was wrong for a reason that has nothing to do with drift: plugin
skills are namespaced and do not override standalone ones, so a user who
installed the plugin and then ran `rekal init` — which the setup flow tells them
to do — ended up with **both** `/rekal:rekal` and `/rekal` loaded. Two identical
descriptions in context every turn, and worse, two different versions: a plugin
tracks this repository's `main`, while an installed binary is whatever version
the user has. Once those diverge the plugin's newer skill describes flags the
user's binary does not implement.

Setup-only makes that state unreachable rather than managed. The recall skill
has one owner, and it is the artifact whose commands it documents.
`TestPlugin_SetupOnly` pins it: the manifest declares a single root skill named
`rekal-setup`, and `skills/`, `references/`, and `scripts/` must not exist under
`plugin/`.

The plugin declares no `version`, so the git SHA is the version. That is
harmless now — setup instructions are slow-moving and version-independent.

## 3. Asking is the gate

Both skills — the plugin's `rekal-setup` and the embedded skill's
`references/setup.md` — hold the same rule above every command: **ask before
installing, ask before initializing.** Installing pipes a script from the
internet into a shell; `rekal init` writes hooks, a branch, and a line in the
user's `CLAUDE.md`. An agent does not make those decisions.

If the user declines, the skill stands down and says memory is not available
here. Silence is a supported answer, as everywhere else in this skill.

`references/setup.md` stays in the embedded skill for the other direction: an
agent that already has the recall skill but hits `command not found` (a broken
`PATH`) or `not initialized`. The tip routes there on exactly those two error
strings, so the `rekal init` path never reads it — the cost to the existing
surface is one line in `SKILL.md`.

## 4. Publishing — two paths, one of them is not a pull request

**Self-hosted (live now).** The repo is its own marketplace:

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

No review, no gatekeeper, updates land when `main` moves.

**Community marketplace (discovery).** Anthropic does not accept plugins over
git. Pull requests against `anthropics/claude-plugins-community` are closed
automatically — the repo is a read-only mirror of an internal review pipeline.
Submission is a form:

- Console (individuals): <https://platform.claude.com/plugins/submit>
- claude.ai (Team/Enterprise, directory-management access):
  <https://claude.ai/admin-settings/directory/submissions/plugins/new>

Run `claude plugin validate ./plugin --strict` before submitting; the pipeline
runs the same check, plus automated safety screening.
Approved plugins are pinned to a commit SHA in the community catalog and CI
bumps the pin as `main` moves. The catalog syncs nightly, so approval and
installability are not the same moment.

The **official** marketplace (`claude-plugins-official`) is curated at
Anthropic's discretion. There is no application, and the form does not feed it.

## 5. Non-goals

- **Shipping the binary in the plugin.** It is platform-specific, embeds a
  model, and the release tarballs run ~165 MB. The plugin ships the skill and
  the instructions to fetch the binary.
- **Setup as a substrate in the recall skill.** Inside the embedded skill,
  setup is a reference page behind an error string. The triage table stays four
  rows. `rekal-setup` is a separate skill because it lives in a separate
  artifact, not because setup earned a substrate.
- **Auto-installing on failure.** An agent that installs software because a
  command errored is a worse agent. The gate is the user.
