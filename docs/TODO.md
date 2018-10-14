# MVP

1. create json APIs 
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