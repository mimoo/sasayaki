# MVP

1. make sure parsing of protobuf structure is fine
2. Should we rename e2e, storage, etc... cryptoState, storageState, hubState, sasayakiState, etc.? 
    - this would be more in line with Disco/Noise/Strobe :D
* move the mutex to sasayaki State?
2. create tests (I think I need to get rid of global vars for tests)
1. create json APIs 
1. encrypt database
2. notification is two-way channel with client->server for "read" notification? and server->client for "new_msg"
1. create tests! can I do a temp database?
    - creating tests will
1. convo
    - title is first message received in a convo?
1. protobuffer client/server 
    - re-use grpc (but how to get pubkey? from RemoteAddress in Disco?)
    - implement my own grpc by just having a 2 byte length header
    - a goroutine is 4,000B. So having an additional 10,000B static array sucks a lot!
1. encryption/decryption
    - should I use a nonce-based approach?
    - if I do that, I should probably do a simple ratchet then
1. creation of threads
    - perhaps I should create new threads via the shared secret + an ephemeral key on our side?
1. create hub in-memory forwarder
1. vue + vuex single-page
5. websockets
6. real db for hub



# After

* check all todos
* generate my own key to start with 0xdeadbeef (2^32 computations)
* yubikey support

# way after

* versioning
* pack daemon to display it in the menu bar or something
* auto update?