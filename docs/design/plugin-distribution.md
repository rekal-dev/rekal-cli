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
├── bin/rekal-install                 ← byte-identical copy of scripts/install.sh
├── skills/install/SKILL.md           → /rekal:install   (once per machine)
├── skills/init/SKILL.md              → /rekal:init      (once per repository)
└── README.md

.claude-plugin/marketplace.json       ← repo root: the catalog listing it
cmd/rekal/cli/skill/skills/rekal/     ← the recall skill — binary only, not shipped here
```

**Two commands, because they have different lifecycles.** Installing the binary
happens once per machine; `rekal init` happens once per repository, and again in
the next repository, and the one after. Collapsing them into a single "setup"
flow assumes setup is a one-time event and the plugin is dead weight afterward.
It isn't: the plugin is user-scoped, so it is loaded in every repo the user
opens, which makes it the natural home for the recurring per-repo action.

Both skills are also model-invoked — `command not found` routes to `install`,
`not initialized` routes to `init`. The `init` skill's description explicitly
tells it **not** to volunteer on a repo that merely lacks `.rekal/`; almost no
repo should be initialized, and a skill loaded everywhere that proposes
initializing everything is a nag.

**The installer is vendored, not fetched.** `bin/` lands on the Bash tool's
`PATH`, so `bin/rekal-install` — a byte-identical copy of `scripts/install.sh`,
pinned by `TestPlugin_VendoredInstaller` — replaces piping a live URL into a
shell. The install logic is then part of the reviewed, SHA-pinned plugin rather
than whatever that URL serves at run time. It still downloads the platform
archive from GitHub Releases, which is unavoidable: the binary embeds a ~134 MB
model, so the release archives run ~160 MB and cannot ship inside a plugin (see
§5).

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
`TestPlugin_SetupOnly` pins it: `plugin/skills/` holds exactly `install` and
`init`, and `skills/rekal`, `references/`, and `scripts/` must not exist under
`plugin/`.

The plugin carries its own `version`, on its own line — `0.1.0`, unrelated to
the binary's `v0.2.x`. They are different artifacts now, and the version says
so. Omitting it would fall back to the git commit SHA and auto-update on every
push, but `claude plugin validate --strict` treats a missing version as an
error, and the submission pipeline runs that check. So: **bump the version in
`plugin/.claude-plugin/plugin.json` whenever the setup skill changes**, or
self-hosted-marketplace users stay pinned to what they installed. The community
catalog pins a commit SHA and re-pins as `main` moves, so that path updates
either way.

## 3. Asking is the gate

Both setup skills hold the same rule above every command: **ask before
installing, ask before initializing.** `rekal init` writes hooks, a branch, and
a line in the user's `CLAUDE.md`; installing puts an executable on their
machine. An agent does not make those decisions.

If the user declines, the skill stands down and says memory is not available
here. Silence is a supported answer, as everywhere else in Rekal.

**Setup lives entirely in the plugin.** The embedded recall skill is untouched by
any of this — byte-identical to what it was before the plugin existed. That is
the cleanest form of the ownership split: the plugin is purely additive, the
binary's skill carries no setup material at all, and no release is required for
the plugin to work. A user who asks to set Rekal up reaches `/rekal:install` or
`/rekal:init`, which are the only surface that exists before the binary does.

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

- **Shipping the binary in the plugin.** The embedded model alone is 140,589,761
  bytes gzipped; release archives run 162–169 MB, across three platforms. GitHub
  rejects any file over 100 MB at push, so it cannot even be committed. LFS does
  not rescue it — a plugin fetch does not run `git lfs pull`, so users would
  receive the 134-byte pointer and fail silently, exactly the trap
  `docs/cloud-agent-setup.md` already documents. And the plugin cache is
  version-keyed (`cache/<marketplace>/<plugin>/<version>/`), so every bump would
  leave another half-gigabyte behind. The plugin ships the 8.5 KB installer
  instead.
- **Setup as a substrate in the recall skill.** Inside the embedded skill,
  setup is a reference page behind an error string. The triage table stays four
  rows. `rekal-setup` is a separate skill because it lives in a separate
  artifact, not because setup earned a substrate.
- **Auto-installing on failure.** An agent that installs software because a
  command errored is a worse agent. The gate is the user.
