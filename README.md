# agent-compose

A name, a job, and the context to do it

![agent-compose and $ acompose, a name, a job, and the context to do it](assets/banner/agent-compose-banner.jpg)

agent-compose compiles the context an agent harness loads. It selects a role,
the personality meld that role carries, the skills that role can see, and the
tool inventory it gets, then materializes one immutable bundle of plain files.
Claude Code, Codex, Goose, and OpenCode all take the same bundle.

The bundle is context and nothing executable. Permissions, runtime facts, and
lifecycle stay with whatever launches the agent, so a role slug travels without
authority attached to it. You can read and diff every file before a run, and
`verify` reports the bundle complete before you do.

## The roster

`roster:core` is the zero-config default, and it ships eight seats. Each one has
a name, a charter, and a meld of two personality traits that shape how it writes
and what it reaches for first.

- 🪢🪨 **Angie** (she) - Platform Engineer - builds and lands the foundational software the rest of the estate is built on. Tenacious and grounded.
- 🛡️🪨 **Vera** (she) - Systems Administrator - operates the real hosted systems and release surfaces. Protective and grounded.
- 🧪🪨 **Evie** (she) - Applied Scientist - measures how agents, models, and inference actually behave on real hardware. Empirical and grounded.
- 🎨🌈 **Delphi** (she) - Frontend Engineer - shapes and builds the surfaces a person navigates. Playful and imaginative.
- 🤿🌈 **Sprite** (they) - Game Developer - ships playable games, the code and the assets and the build that carries both. Immersed and imaginative.
- ✂️🔭 **Portia** (they) - Portfolio Director - decides what the portfolio does next, and carries each decision to its gate. Decisive and outward.
- 🕯️🔭 **Gem** (they) - Developer Advocate - turns real work and audience evidence into accurate content and informed commitments. Warm and outward.
- 🪞🪢 **Cassandra** (they) - AI Risk Analyst - **archived** - assessed whether an AI system was fit to ship, named the exclusions, and stood behind the file that proved it. Suspicious and tenacious.

Every seat melds one signature trait with one bond it shares with a sibling, so
the eight signature traits are distinct and the bonds group them: the three
builders share 🪨 grounded, the two makers share 🌈 imaginative, and the two
outward-facing seats share 🔭 outward. The analyst bonds instead to 🪢
tenacious, the platform seat's own signature, so eight signatures and three
shared bonds is the eleven the catalogue lists.

Each personality carries a colour, an emblem, a motif, and a body written in
prose, which is where the creature art comes from and what a voice melds along
with the role's own. See [docs/identity.md](docs/identity.md) and
[docs/personality.md](docs/personality.md). `just palette-serve` renders the
whole catalogue locally.

Selection is exclusive. An external person package contributes its own roles,
seats, personality definitions, and evaluation context, and it replaces
`roster:core` wholesale rather than merging with it. An `external-only` policy
makes that boundary fail closed across the machine. See
[docs/person-packages.md](docs/person-packages.md).

## Install

```sh
brew tap coilyco-flight-deck/tap https://forgejo.coilysiren.me/coilyco-flight-deck/homebrew-tap
brew install coilyco-flight-deck/tap/agent-compose
```

```powershell
scoop bucket add coilyco-flight-deck https://forgejo.coilysiren.me/coilyco-flight-deck/scoop-bucket
scoop install coilyco-flight-deck/agent-compose
```

Both also install `acompose`, the compose verb directly, and both install the
roster beside the binary. A raw binary from
[Forgejo releases](https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose/releases)
needs two assets rather than one, because the binary carries no roster:

```sh
curl -fL .../download/agent-compose-roster.tar.gz | tar xz -C ~/.agent-compose
```

Skip that and the first `catalog roles` reports `no roster is mounted`, naming
every path it tried. Releases attach darwin-arm64, linux-amd64, linux-arm64 and
windows-amd64. From source, `just install` builds into `GOBIN` and exports the
roster path for you. `agent-compose version` reports the build you are running.

## Use it

```sh
acompose                              # converge the host from the local catalogue
acompose -- <command>                 # refresh context, then exec the command
agent-compose launch <role> <harness> # launch one harness with an assigned role
```

New here? [guides/quickstart.md](guides/quickstart.md) goes from an empty
terminal to a composed role, and the role guides beside it cover when to reach
for each seat.

`--reapply` rewrites the host compose layout even when it is already current,
and `--verbose` prints every `source => destination` mapping it places.
`--explain` adds the role briefing, the credits, the expression vocabulary,
and the full decision tree to the summary.

When you want to see what a bundle actually decided rather than trust it,
`describe` renders its stored decision tree, `diff` reports the semantic change
between two bundles, and `verify` checks one is complete and safe to consume.
Composition adapters can project a verified bundle into an empty staged home
and wrap it in their own schema. See
[docs/staged-home.md](docs/staged-home.md).

## The board comes from the roster

The evaluation board is derived from the roster rather than written beside it.
housecast's `evalkit.matrix` reads the roster and prints the cases it implies:
boundaries and their owners produce the pairs, adjacency produces the role-fit
targets, and each role's meld produces the personality cases. Add a boundary,
flip an adjacency edge, or swap a personality, and the challenge list moves with
it. Every case corresponds to something in the roster, and every roster change
moves what gets tested.

The hard cases are generated on purpose. Role adjacency names each role's two
likeliest absorptions, and those reasons become the descriptors a generator uses
to build exactly the confusion a seat is most at risk of.

Three parties and none of them holds two seats: a generator authors the cases, a
subject answers them through Agent Proxy, and a human grades them. The grading
half ships separately as `housecast grade`, so it holds no runner and no model
client, and grading never spends a token or touches a deployed system.
Details in [docs/evaluation.md](docs/evaluation.md).

### What the board needs, and what runs without it

Two pieces of the eval half come from outside this repository. housecast is
pinned from Forgejo by tag, and the subject answers through Agent Proxy, an
internal transport, so running the board yourself means supplying a model
transport in its place.

The composer stands on its own. The bundle, the roster, the cascade, `describe`,
`diff`, and `verify` all work with the eval stack absent, which is the half you
get from `brew install` alone.

## Development

Dev verbs are recipes in the [justfile](justfile). `just test` runs the Go tests
plus the full hook sweep. `just smoke` builds the real `acompose` entry point and
converges an isolated temporary home twice, covering roster, cascade, skills,
load points, and idempotence without touching live host state or the network.
`just palette-serve` starts the local personality palette explorer.

Every push to canonical `main` validates and publishes the next minor release.
Forgejo is canonical and the GitHub mirror is verified.

## License

MIT. See [LICENSE](LICENSE).

## See also

- [guides/quickstart.md](guides/quickstart.md) - install, configure, converge.
- [AGENTS.md](AGENTS.md) - agent-facing operating rules.
- [docs/FEATURES.md](docs/FEATURES.md) - inventory of what ships today.
- [docs/architecture.md](docs/architecture.md) - the composition boundary.
- [docs/ownership.md](docs/ownership.md) - who owns which boundary, and why.
- [docs/role-selection.md](docs/role-selection.md) - role-scoped providers and assigned bundles.
- [docs/identity.md](docs/identity.md) - names, emblems, bodies, and how a voice melds.
- [docs/personality.md](docs/personality.md) - the personality catalog and palette.
- [docs/evaluation.md](docs/evaluation.md) - the generator, subject, and grader split.
- [docs/release.md](docs/release.md) - the automatic Forgejo release pipeline.
- [justfile](justfile) - development recipes.
- [.ward/ward.yaml](.ward/ward.yaml) - catalog metadata only.
