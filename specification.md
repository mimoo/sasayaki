# Sasayaki specification

## Server

* The server uses TLS
* it accepts protobuf messages
* they contain metadata + length content + content
* content is a disco handshake at first (between the users)
    - and then encrypted blobs

## Authentication

* NoiseIK

## Sending a message

* if you've never added the person, initiate a Noise handshake
* 
```
{to: Alice, content:first_Noise_handshake_msg}
```