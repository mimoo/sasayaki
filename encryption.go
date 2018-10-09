//
// Encryption Manager
// ==================
//
// Each conversation has two states:
//
// * Alice -> Bob state
// * Bob -> Alice state
//
// When encrypting or decrypting a message, one of these states needs to be fetched from the database
//
// notes:
// - id and convo id are unsigned 64bit integers. We can generate them randomly, it shouldn't matter
// 	since we care about the tuple {id, convoid, toAddress, fromAddress} which has very little chance of colliding

package main

type encryptionManager struct {
}

func initEncryptionManager(id, convoId uint64, fromAddress string, content []byte) {
	// if id = 1 -> new convo, derive new key from c2
	// else:
	// sessKey, ok := ds.GetSessionKey(convoId)
	// if !ok {
	// 	we have a problem huston
	// }
}
