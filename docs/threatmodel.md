# Threat Model and Rational

(Sasayaki adopts the Internet threat model [RFC 3552](https://tools.ietf.org/html/rfc3552) and therefore assumes that the attacker has complete control over the network.) -> maybe not

Sasayaki adopts the [honest by curious](https://crypto.stanford.edu/pbc/notes/crypto/sfe.html) threat model, where a hub reliably delivers messages in order, but still might try to spy on users.

Sasayaki or NCC Group Messenger was designed as an end-to-end encrypted messaging application for a specific work environement (NCC Group):

* no multi-device support: a single work station hosting the keys
* no self-revocation: as team member are supposed to be closely in contact, they can revoke each other's key
    - what if server doesn't propagate these revoke messages?
* no asynchronous messaging: contacts must interactively add each other before starting to exchange messages
* no **future secrecy**: devices are not phones and are protected under full disk encryption. If a device is compromise at a point t, we assume that this is already a catastrophe.
     - perhaps I could kind of introduce future secrecy when we start a new thread 
        + it's encrypted using something produced with c1
* no **forward secrecy**: most people keep logs on their hard drive, if you get hacked then you will lose these logs as well. What's the point protecting against forward secrecy?
    - actually I have forward secrecy of threads, but I don't have it for specific messages in thread
    - I could do a RATCHET after each message in a thread (?)
* no protection against denial of service: a hub who decides to stop relaying messages is not in scope
* no gossip protocols, so if the server refuses to tell you that someone's key was revoked... that's bad
* if we have notice of receipt, we won't tie any security to it (too many messages)

overall: the goal is to achieve something better than PGP email encryption inside of a company. Not to achieve maximum paranoid settings which are mostly useless

* two types of secrets: the keys and the data