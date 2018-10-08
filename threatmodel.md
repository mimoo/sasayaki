# Threat Model and Rational

Sasayaki or NCC Group Messenger was designed as an end-to-end encrypted messaging application for a specific work environement (NCC Group):

* no multi-device support: a single work station hosting the keys
* no self-revocation: as team member are supposed to be closely in contact, they can revoke each other's key
    - what if server doesn't propagate these revoke messages?
* no asynchronous messaging: contacts must interactively add each other before starting to exchange messages
* no future secrecy: devices are not phones and are protected under full disk encryption. If a device is compromise at a point t, we assume that this is already a catastrophe.
* no protection against denial of service: a hub who decides to stop relaying messages is not in scope
* no gossip protocols, so if the server refuses to tell you that someone's key was revoked... that's bad