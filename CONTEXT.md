# brewupdater

A Go tool that shows which Homebrew installs are outdated and lets the user quickly pick which ones to upgrade.

## Language

**Outdated package**:
An installed Homebrew formula or cask whose installed version is behind the latest version available in its tap.
_Avoid_: stale install, old version

**Upgrade**:
Moving a selected **Outdated package** to the latest version available in its tap (what `brew upgrade <name>` does).
_Avoid_: update (in brew, `brew update` means refreshing tap metadata, not upgrading packages)

**Selection**:
The set of **Outdated packages** the user picked to **Upgrade** in one run.

**Favorite**:
A package name listed in the user's config, which is shown first (in config order) and preselected when it is outdated.
_Avoid_: preferred, starred

**Pinned package**:
An **Outdated package** the user pinned with `brew pin`; it is shown but can never be part of a **Selection**.

## Relationships

- A **Selection** contains zero or more **Outdated packages**
- The initial **Selection** is every outdated **Favorite** that is not a **Pinned package**
- A **Favorite** that is not outdated is not shown
- Each **Outdated package** in a **Selection** is **Upgraded** to latest; choosing an arbitrary target version is out of scope

## Flagged ambiguities

- "choose which version to update" — resolved: the user chooses *which packages* to **Upgrade**, not a target version. Every **Upgrade** goes to latest.
- "update" vs "upgrade" — in Homebrew `update` refreshes metadata and `upgrade` changes installed packages; this project uses **Upgrade** for the latter.
