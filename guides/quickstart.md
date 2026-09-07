# Quickstart

From an empty terminal to an agent that knows which seat it is sitting in.
Nothing here assumes a checkout of this repository, a package registry account,
or any file you already have. Every command below was run against a released
binary in a scratch home before it was written down.

## What you are about to build

Three files, in three places, and one command that reads them.

* `~/.agent-compose/roster` - the seats themselves - eight roles, their
  personalities, and the boundaries between them. The binary does not carry
  this. It mounts it.
* `~/.agent-compose/agent-compose.yaml` - your host configuration - which
  doctrine files to compose, which repositories are in play, and where the
  composed context should land.
* `<your repo>/.agents/roles.kdl` - which roles this repository actually offers,
  and what extra context each one gets.

Miss any of the three and the tool tells you which one and where it looked. The
error messages are the real reference page. Read them rather than guessing.

## 1. Install

The package managers install the binary and the roster together, which is the
path with the fewest steps.

```sh
brew tap coilyco-flight-deck/tap https://forgejo.coilysiren.me/coilyco-flight-deck/homebrew-tap
brew install coilyco-flight-deck/tap/agent-compose
```

```powershell
scoop bucket add coilyco-flight-deck https://forgejo.coilysiren.me/coilyco-flight-deck/scoop-bucket
scoop install coilyco-flight-deck/agent-compose
```

If you take a raw binary from the
[releases](https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose/releases)
instead, take two assets, not one. The binary alone has no seats in it:

```sh
curl -fLo ~/.local/bin/agent-compose \
  https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose/releases/latest/download/agent-compose-darwin-arm64
chmod +x ~/.local/bin/agent-compose

mkdir -p ~/.agent-compose
curl -fL https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose/releases/latest/download/agent-compose-roster.tar.gz \
  | tar xz -C ~/.agent-compose
```

The tarball unpacks to `roster/`, so that lands at `~/.agent-compose/roster`,
which is the second place the binary looks. Check it took:

```sh
agent-compose catalog roles
```

Eight lines, one per seat. If instead you get `no roster is mounted`, the
message lists every directory it tried, in order. Put the roster in one of them,
or point `AGENT_COMPOSE_ROSTER` at wherever you unpacked it.

## 2. Read the seats before you pick one

```sh
agent-compose catalog roles        # purpose, personalities, colour
agent-compose catalog seats        # the name each role answers to, per harness
agent-compose catalog boundaries   # who owns what, and who defers
```

The boundary table is the one worth staring at. Every cell is `OWNS`, `scope`,
or `defers`, and it is the whole design in one screen: exactly one seat owns
each boundary, a few hold a named slice of it, and the rest hand the work over
rather than doing it badly.

```
boundary                     platform  sysadmin  science  frontend  gamedev  director  advocate  underwriter
modify-live-backend          scope     OWNS      defers   defers    scope    defers    defers    defers
suggest-external-comms       defers    defers    defers   scope     scope    defers    OWNS      defers
seek-external-validation     scope     defers    defers   defers    defers   OWNS      scope     scope
build-foundational-software  OWNS      scope     scope    defers    defers   defers    defers    defers
```

Which column you want is the subject of the role guides, one per seat, listed
at the bottom of this page.

## 3. Configure the host

Write `~/.agent-compose/agent-compose.yaml`. This is the smallest one that runs:

```yaml
sources:
  - /Users/you/doctrine.md
roots:
  - /Users/you/.agent-compose/sources
operating_context:
  - acme/widget
load_points:
  claude: /Users/you/code/acme/widget/CLAUDE.md
```

* `sources` - your own doctrine, composed in ahead of the seats. One file is
  enough, and an empty project can point at a file containing one sentence.
* `roots` - where composed intermediate sources are written. Keep it inside the
  state directory unless you have a reason not to.
* `operating_context` - `owner/repository` for each repository in play. These
  resolve under `$PROJECTS_ROOT`, so set that to the directory your checkouts
  live in.
* `load_points` - where the finished context file lands for each harness. This
  is the path your agent will actually read.

Every one of these is required in the sense that leaving it out produces a
specific complaint rather than a silent partial run. Omitting
`operating_context` gives you `agent-compose.yaml must declare operating_context
repositories` and stops.

## 4. Declare the roles your repository offers

In the repository you named, write `.agents/roles.kdl`:

```kdl
roles {
    role "platform" {
        composed-skill "widget-method"
    }
    role "science" {
        composed-skill "widget-method"
    }
}
```

This is the step people skip, and it is the one that decides which seats exist
here. A role absent from this file cannot be launched in this repository, even
though `catalog roles` lists it, because the roster says a seat is possible and
`roles.kdl` says it is offered. Ask for one you did not declare and you get
`repository plan has no role "science", available roles: platform`.

A role body may stay empty. `composed-skill` names a directory under
`.agents/composed/<name>/COMPOSED.md`, which is repository doctrine only that
seat should read. Ordinary skills under `.agents/skills/` are discovered without
being listed. Full grammar in [KDL contracts](../docs/kdl-contracts.md).

## 5. Converge

```sh
acompose --verbose
```

`acompose` is the compose verb directly, installed alongside `agent-compose`.
Without `--verbose` a successful run says nothing at all, which is correct for
something you will end up running from a shell hook.

The verbose transcript names every `source => destination` it placed, then ends
with two counts:

```
cascade outputs=2 load-points=2 repository-plan=1 changed=5
skills  managed=48 load-points=2 verified=0 linked=48 removed=0 preserved=0
```

Read your load point. It opens with your own doctrine, then the personality
invariant, then every seat this deployment offers.

## Host convergence is not the only shape

That composed file carried all eight seats: 52,912 bytes in the run above, one
charter after another, so the agent can see the whole roster and switch inside
it. That is the right shape for an interactive session where you have not
decided yet.

When you have decided, assign the role instead and carry one:

```sh
agent-compose bundle materialize --role science --harness claude --out ./bundles
agent-compose launch science claude
```

The same roster produced 7,154 bytes for that bundle, one charter, two skills.
Seven times less context, and a seat that cannot switch out of its own charter
because nothing else is in the file. Assigned beats inferred whenever you
already know the answer.

Inspect one before you trust it:

```sh
agent-compose describe ./bundles/<id>   # the decision tree it stored
agent-compose verify ./bundles/<id>     # complete and safe to consume
agent-compose diff <old> <new>          # what actually changed between two
```

## The role guides

One page per seat. Each carries what that seat owns, what it holds a slice of,
what it hands over, and the adjacent seat the roster says it drifts toward.

* [platform](platform.md) - Platform Engineer. Owns the foundational software
  every other seat stands on.
* [sysadmin](sysadmin.md) - Systems Administrator. Owns every change to a
  running hosted system.
* [science](science.md) - Applied Scientist. Owns nothing on purpose, and
  produces the evidence the others act on.
* [frontend](frontend.md) - Frontend Engineer. Builds the surfaces a person
  navigates.
* [gamedev](gamedev.md) - Game Developer. Ships the playable thing, code and
  assets and build together.
* [director](director.md) - Portfolio Director. Owns reaching outside for
  evidence, and carries each decision to its gate.
* [advocate](advocate.md) - Developer Advocate. Owns everything addressed
  outward to a reader.

The eighth seat, `underwriter`, ships in `roster:core` and appears in the
boundary table above. It has no guide here yet.
