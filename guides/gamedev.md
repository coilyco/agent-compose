# gamedev

Game Developer. Whale-Dragonfly.

**Purpose** - ship playable games: the code, the assets, and the build that
carries both.

**Meld** - immersed and imaginative. It plays the thing rather than reading
about it, and judges by what the loop feels like from inside.

**Harnesses** - claude, codex, openhands. Supports the commodity tier.

Print the seat before you read about it:

```sh
agent-compose overlay --role game-dev --seat claude
```

```
🤿 🌈 Whale-Dragonfly
gamedev / available
immersed + imaginative
#2980fe
```

Then take it:

```sh
agent-compose launch gamedev claude
```

## What it owns

Nothing. Like [frontend](frontend.md), this seat produces a finished artifact
for a person rather than a platform other seats stand on, so it holds scopes
rather than a boundary.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role game-dev --harness claude --out ./bundles
agent-compose describe ./bundles/0fa273eb460142e2
agent-compose verify ./bundles/0fa273eb460142e2
```

```
bundle 0fa273eb460142e2 // gamedev/immersed+imaginative // native-skills // 7523 body bytes
profile
  ✓ boundary modify-live-backend          role "game-dev" holds within a scope boundary
  ✓ boundary suggest-external-comms       role "game-dev" holds within a scope boundary
  ✓ boundary build-foundational-software  role "game-dev" defers boundary
  ✓ boundary seek-external-validation     role "game-dev" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

Two, and both are unusually specific.

* `modify-live-backend`, scoped to a local world, server, or save it launched
  itself, plus routine operation of a game server it already runs: mod
  sync, restart, config reload, world backup. Never provisioning a new server,
  changing its topology or capacity, or a first deployment.
* `suggest-external-comms`, scoped to in-game text, item descriptions, tooltips,
  and mod documentation. Never a patch announcement, store description, or post
  about the game.

Read the first one carefully. It is the most generous slice any non-owning seat
holds anywhere in the roster, and it exists because a seat whose work requires
a running world should not have to hand over every restart. The line inside it
is between operating something that exists and changing what exists.

## What it defers

`build-foundational-software` and `seek-external-validation`. The engine, the
shared tooling, and the question of whether this game is the right game are all
elsewhere.

## Reach for it when

* A mod needs writing, or an existing one broke against a game update.
* A game loop needs tuning by someone willing to play it repeatedly.
* An asset pipeline or build is failing between the editor and the artifact.
* A crash needs reproducing in an actual session rather than reasoned about.
* A world's state needs understanding before anything is done to it.

## The tell that you picked wrong

* **Toward sysadmin** - operating the hosted world rather than the local one it
  is free to run. This is the one that matters most here, because the scope
  grant makes it easy to walk into by degrees. If the next command would
  provision, resize, or first-deploy, that is [sysadmin](sysadmin.md).
* **Toward frontend** - polishing the surface instead of playing the loop
  underneath it. A menu that looks better has not made the game better.

## The chain it sits in

The two scope grants make this seat unusually self-sufficient, which is the
point: a gamedev seat handing over every server restart would spend its session
waiting. Everything past its edge goes elsewhere, the hosted world to
[sysadmin](sysadmin.md) and the engine underneath to
[platform](platform.md).

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md).

## Three prompts to start with

1. The mod broke against this week's game update. Reproduce it in an actual
   session and fix it.

2. Sync the mods, restart the world, and take a backup before you do.

3. Write the item descriptions and tooltips for the twelve new recipes.

**And one it would hand back.** Stand up a second server. Operating the world
you already run is inside the scope, provisioning a new one is sysadmin's.
