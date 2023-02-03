package srp

import (
	"crypto/rand"
	"io"
	"math/big"
)

// GenKey generate a random key
func GenKey() ([]byte, error) {
	bytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// getx compute the intermediate value x as a hash of salt, identity and password.
// getx return the user secret as a big int.
func getx(params *Params, salt, I, P []byte) *big.Int {
	var ipBytes []byte
	ipBytes = append(ipBytes, I...)
	ipBytes = append(ipBytes, []byte(":")...)
	ipBytes = append(ipBytes, P...)

	hashIP := params.Hash.New()
	hashIP.Write(ipBytes)

	hashX := params.Hash.New()
	hashX.Write(salt)
	hashX.Write(hashToBytes(hashIP))

	return hashToInt(hashX)
}

// getMultiplier calculate the SRP-6 multiplier
func getMultiplier(params *Params) *big.Int {
	hashK := params.Hash.New()
	hashK.Write(padToN(params.N, params))
	hashK.Write(padToN(params.G, params))
	return hashToInt(hashK)
}

// getK Compute the shared session key K from S
func getK(params *Params, S []byte) []byte {
	hashK := params.Hash.New()
	hashK.Write(S)
	return hashToBytes(hashK)
}

// getu hashes the two public messages together, to obtain a scrambling parameter "u"
// which cannot be predicted by either party ahead of time. This makes it safe to
// use the message ordering defined in the SRP-6a paper, in which the server reveals
// their "B" value before the client commits to their "A" value.
func getu(params *Params, A, B *big.Int) *big.Int {
	hashU := params.Hash.New()
	hashU.Write(A.Bytes())
	hashU.Write(B.Bytes())

	return hashToInt(hashU)
}

func getM1(params *Params, A, B, S []byte) []byte {
	hashM1 := params.Hash.New()
	hashM1.Write(A)
	hashM1.Write(B)
	hashM1.Write(S)
	return hashToBytes(hashM1)
}

func getM2(params *Params, A, M, K []byte) []byte {
	hashM1 := params.Hash.New()
	hashM1.Write(A)
	hashM1.Write(M)
	hashM1.Write(K)
	return hashToBytes(hashM1)
}
