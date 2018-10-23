# The Sasayaki Messaging Protocol

## Introduction 

Sasayaki responds to these needs:

* a secure messaging application for highly secretive operations
* large companies that require a good key distribution system based on trust between employees
* no protection against hiding identities or relation graphs

## Infrastructure

* The hub's role is to forward messages between identities
* identities are public keys
* it accepts protobuf messages?
* they contain metadata + length content + content
* the hub uses disco to authenticate users
* users use disco between them to secure their conversations

## Authentication to the hub

To connect to the hub, Noise's IK handshake pattern is used. This has the benefit of using the client's public key as identity.

```
IK:
  <- s
  ...
  -> e, es, s, ss
  <- e, ee, se
```

## Adding a contact

Sasayaki is a **synchronous** messaging application, meaning that no messages can be sent unless an interactive handshake has been performed. Ways to create asynchronous messaging application exist but are technically complex to design and implement, see [X3DH](https://signal.org/docs/specifications/x3dh/) and its 100 one-time prekeys, its pre-keys and its keys :)

Alice can add Bob as a contact by creating a unique conversation with him. To do this, a Noise IK handshake message is started with the recipient:

```
{
    to: AlicePublicKey,
    content: [e, E(s), payload1]
}
```

* `payload1` contains the nickname Alice wants Bob to see

The conversation is the locked until Bob performs his end of the handshake:

```
{
    to: BobPublicKey,
    content: [e, payload2]
}
```

* `payload2` contains the nickname Bob wants Alice to see

## Public profiles and Trust in a contact

* identites have public profiles
* these public profiles contain signatures over these keys from other people
* whenever we look at a profile, these signatures are retrieved and checked against who we know
    - not to do it too often we can cache these results?

* Alice's signature over Bob identity looks like this:

```
sign({nickname_len, "Bob", len_mean, "mean of verification", bob_public_key, date of signing, alice_public_key})
```

    - nickname_len + "Bob": needed otherwise ambiguous
    - alice_public_key: because if we don't have the identity of the signer, there could be a DSKS attack?
    - len_mean + mean of verification: could be facebook, irl, twitter, etc... we don't want to restrict users but we can limit chars

* we could also, theoretically, show two degrees of trust, but computationally intensive?

* we are not protecting against profiles and relationship graphs
    - goal is entreprise secure messaging, where identities need to be shared



## Sending a message

If the recipient is a new contact, he/she must first accept by completing the handshake. This is done automatically when they come online.

```
{to: Alice, content:first_Noise_handshake_msg}
```

* I should be able to start several conversations (like email) after I added someone:
  - perhaps adding someone just creates two "shared secret ratchet (ssr)"
    + I can export it after the handshake
  - everytime I create a new thread, I take my direction's "ssr" and I hkdf it to create
    + the next one
    + the conversation's secret
* What happens if someone lose their keys?
  - I should get notified (revocation by broadcasters/companies and friends?)
  - all my convo with that person should be blocked (or at least it should say "1 companies + 2 friends have revoked that key")
  - all new convo should have the warning as well
  - so should I have an option to update someone's key?

## Group chat

* see MLS for building the group chat key?
* but I want transcript consistency?

## Reddit like group chat

* that would be an interesting feature, kanban or reddit-like or github issues-like
* actually since this is limited in number of people, github issues-like would be the best

## Broadcast

* to facilitate onboarding of new employee, we can have "broadcasters" or "companies"
* example: NCC Group account, that pretty much just signs {public keys + nickname}
* this can allow us to browse a directory of employee that has been onboarded by an official public key
* we can obtain this public key during our own onboarding (we trust onboarding obviously)
* we can follow particular companies/broadcasters to access their broadcasted contact lists
* these could also be further divided by teams, or cross signed?


# Details

* we use protobuf, why? It seems way faster than json. See first section of https://developers.google.com/protocol-buffers/docs/gotutorial as well

# API

## GET

* get_messages: get all messages pending for myself
* 

## POST

* message_read_ack: tells the forwarder that it can safely delete a message (otherwise it will be deleted after X days)
  - we can send several ack in the message msg
* send_message: send a message to someone
* 

# States

## SasayakiState

* initialized
* mutex?
* myAddress
* encryptionState
* storageState
* hubState

functions:

* `initSasayakiState()`: this function does the following things:
  - initEncryptionState(keyPair)
  - initstorageState()
  - sets initialized to true

## EncryptionState

e2e encryption

* keyPair

functions:

* `initEncryptionState(keyPair)`:
* encryptMessage()
* decryptMessage()
* createNewConvo()
* createConvoFromMessage()
* addContact()
* acceptContact()
* finishAddContact()

## StorageState

The Storage state is used to retrieve and update the local database. It is database agnostic, although the main implementation of Sasayaki (NCC Group Messenger) uses sqlite. It is supposed to transparently encrypt/decrypt rows from the database `decrypt(key=k, nonce=row.id, data=row.others, ad=table.name)` with `k = AD(passphrase)|BLIND(16)|OPRF()|UNBLIND`. It has the following values:

* databaseAddress

schema?

functions:

* `initStorageState(storageKey)`:
* TKTK

## HubState

The HubState is used to communicate (using protobuffers) with the hub. This is the core spec which doesn't specify the protobuffers requests. Check the NCC Group Messenger spec for actual the protobuffers objects

* hubAddress: the address of the hub
* hubPublicKey: the public key of the hub
* messagestoAck: an array of messages that the server can delete (do we really need this?)

It responds to the following functions:

* `InitHubState()`:
* `sendMessage()`:
* `getNextMessages(numberMessages)`: requests the next "numberMessages". 





