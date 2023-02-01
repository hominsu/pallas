package srp

import (
	"bytes"
	"errors"
	"math/big"
)

type Server struct {
	Params   *Params
	Verifier *big.Int
	Secret   *big.Int
	B        *big.Int
	M1       []byte
	M2       []byte
	K        []byte
	u        *big.Int
	s        *big.Int
}

func NewServer(params *Params, verifier []byte, secret []byte) *Server {
	multiplier := getMultiplier(params)
	v := intFromBytes(verifier)
	se := intFromBytes(secret)
	Bb := getB(params, multiplier, v, se)
	B := intFromBytes(Bb)
	return &Server{
		Params:   params,
		Secret:   se,
		Verifier: v,
		B:        B,
	}
}

func (s *Server) ComputeB() []byte {
	return intToBytes(s.B)
}

func (s *Server) ComputeK() []byte {
	return s.K
}

func (s *Server) SetA(A []byte) error {
	AInt := intFromBytes(A)
	U := getu(s.Params, AInt, s.B)
	S, err := serverGetS(s.Params, s.Verifier, AInt, s.Secret, U)
	if err != nil {
		return err
	}

	s.K = getK(s.Params, S)
	s.M1 = getM1(s.Params, A, intToBytes(s.B), S)
	s.M2 = getM2(s.Params, A, s.M1, s.K)

	s.u = U               // only for tests
	s.s = intFromBytes(S) // only for tests

	return nil
}

func (s *Server) CheckM1(M1 []byte) ([]byte, error) {
	if !bytes.Equal(s.M1, M1) {
		return nil, errors.New("m1 mismatch")
	}
	return s.M2, nil
}

// getB returns the server's public value(B) which carried by server key exchange message.
// The server calculates this value as B = k*v + g^b % N, where b is a random number.
func getB(params *Params, multiplier, V, b *big.Int) []byte {
	gModPowB := new(big.Int)
	gModPowB.Exp(params.G, b, params.N)

	vMullK := new(big.Int)
	vMullK.Mul(multiplier, V)

	leftSide := new(big.Int)
	leftSide.Add(vMullK, gModPowB)

	final := new(big.Int)
	final.Mod(leftSide, params.N)

	return padToN(final, params)
}

// serverGetS returns TLS premaster secret
func serverGetS(params *Params, V, A, S2, U *big.Int) ([]byte, error) {
	ALessThan0 := A.Cmp(big.NewInt(0)) <= 0
	NLessThanA := params.N.Cmp(A) <= 0
	if ALessThan0 || NLessThanA {
		return nil, errors.New("invalid client-supplied 'A', must be 1..N-1")
	}

	result1 := new(big.Int)
	result1.Exp(V, U, params.N)

	result2 := new(big.Int)
	result2.Mul(A, result1)

	result3 := new(big.Int)
	result3.Exp(result2, S2, params.N)

	result4 := new(big.Int)
	result4.Mod(result3, params.N)

	return padToN(result4, params), nil
}
