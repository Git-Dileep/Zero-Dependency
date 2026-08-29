package ratchet

import (
	"crypto/hmac"
	"crypto/sha256"
)

type Key32 = [32]byte

type RatchetState struct {
	RootKey  Key32
	ChainKey Key32
	Epoch    uint32
}

// Advance derives the next chain key and message key from the current chain key,
// and securely overwrites the old chain key in memory.
func (rs *RatchetState) Advance() (messageKey Key32, err error) {
	// messageKey = HMAC-SHA256(ChainKey, 0x01)
	hmacMsg := hmac.New(sha256.New, rs.ChainKey[:])
	hmacMsg.Write([]byte{0x01})
	copy(messageKey[:], hmacMsg.Sum(nil))

	// nextCK = HMAC-SHA256(ChainKey, 0x02)
	hmacNext := hmac.New(sha256.New, rs.ChainKey[:])
	hmacNext.Write([]byte{0x02})
	var nextCK Key32
	copy(nextCK[:], hmacNext.Sum(nil))

	// Overwrite the old ChainKey bytes in memory before replacing it.
	for i := range rs.ChainKey {
		rs.ChainKey[i] = 0
	}
	rs.ChainKey = nextCK

	return messageKey, nil
}

// Reseed resets the state with a new root key and chain key for a new epoch.
func (rs *RatchetState) Reseed(newRootKey, newChainKey Key32) {
	// Zero out old keys
	for i := range rs.RootKey {
		rs.RootKey[i] = 0
	}
	for i := range rs.ChainKey {
		rs.ChainKey[i] = 0
	}
	
	rs.RootKey = newRootKey
	rs.ChainKey = newChainKey
	rs.Epoch++
}
