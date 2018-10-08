# Gossip

## Problem

these are the attacks that can be done by a rogue hub:

* an organization refusing to show revokations on a user

## Proposed Solutions

**Merkle Tree**. An organization is an append only merkle tree with leafs containing:

* members and their signatures
* revokations

now people just need to gossip about the root of that tree and make sure they have all the leafs (members + revokations)

**Blockchain**. An organization is a succession of blocks containing the same as the merkle tree leafs content.

For this to work, we would need to wait to batch include them in a block OR we would have one block per member/revokation. If the overhead of a block is large, this could be annoying (at least a hash).

## Discussions

Do we need a Merkle Tree? We need to collect all the solutions. The Merkle Tree would allow new users to:

* get the list of members and revokations
* compute the root themselves

and current users would:

* receive updates about new members and revokation
* re-compute the root

For this to work, an organization needs to have a "meta" channel open with all of its users 

- do we want to include people outside of an organization?
- do we want to have users w/o organizations?
- do we want to have more than one organization?

