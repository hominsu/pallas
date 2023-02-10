package srp

import (
	"bytes"
	"errors"
	"math/big"
)

type Client struct {
	Params     *Params
	Multiplier *big.Int
	Secret     *big.Int
	A          *big.Int
	X          *big.Int
	u          *big.Int
	s          *big.Int
	M1         []byte
	M2         []byte
	K          []byte
}

func NewClient(params *Params, salt, identity, password, secret []byte) *Client {
	multiplier := getMultiplier(params)
	se := intFromBytes(secret)
	A := intFromBytes(getA(params, se))
	x := getx(params, salt, identity, password)

	return &Client{
		Params:     params,
		Multiplier: multiplier,
		Secret:     se,
		A:          A,
		X:          x,
	}
}

func (c *Client) ComputeA() []byte {
	return intToBytes(c.A)
}

func (c *Client) SetB(Bb []byte) error {
	B := intFromBytes(Bb)
	u := getu(c.Params, c.A, B)
	S, err := clientGetS(c.Params, c.Multiplier, c.X, c.Secret, B, u)
	if err != nil {
		return err
	}

	c.K = getK(c.Params, S)
	c.M1 = getM1(c.Params, intToBytes(c.A), Bb, S)
	c.M2 = getM2(c.Params, intToBytes(c.A), c.M1, c.K)

	c.u = u               // Only for tests
	c.s = intFromBytes(S) // Only for tests

	return nil
}

func (c *Client) ComputeM1() ([]byte, error) {
	if c.M1 == nil {
		return nil, errors.New("m1 is nil")
	}
	return c.M1, nil
}

func (c *Client) ComputeK() []byte {
	return c.K
}

func (c *Client) CheckM2(M2 []byte) error {
	if !bytes.Equal(c.M2, M2) {
		return errors.New("m2 mismatch")
	}
	return nil
}

// getA returns the client's public value(A) which carried by client key exchange message.
// The client calculates this value as A = g^a % N, where a is a random number.
func getA(params *Params, a *big.Int) []byte {
	ANum := new(big.Int)
	ANum.Exp(params.G, a, params.N)
	return padToN(ANum, params)
}

// ComputeVerifier returns a verifier that is calculated as described in Section 3 of [SRP-RFC].
// The verifier (v) is computed based on the salt (s), user name (I), password (P),
// and group parameters (N, g).
//
// x = H(s | H(I | ":" | P)), v = g^x % N
func ComputeVerifier(params *Params, salt, identity, password []byte) []byte {
	x := getx(params, salt, identity, password)
	vNum := new(big.Int)
	vNum.Exp(params.G, x, params.N)
	return padToN(vNum, params)
}

// clientGetS returns TLS premaster secret
func clientGetS(params *Params, k, x, a, B, u *big.Int) ([]byte, error) {
	BLessThan0 := B.Cmp(big.NewInt(0)) <= 0
	NLessThanB := params.N.Cmp(B) <= 0
	if BLessThan0 || NLessThanB {
		return nil, errors.New("invalid server-supplied 'B', must be 1..N-1")
	}

	result1 := new(big.Int)
	result1.Exp(params.G, x, params.N)

	result2 := new(big.Int)
	result2.Mul(k, result1)

	result3 := new(big.Int)
	result3.Sub(B, result2)

	result4 := new(big.Int)
	result4.Mul(u, x)

	result5 := new(big.Int)
	result5.Add(a, result4)

	result6 := new(big.Int)
	result6.Exp(result3, result5, params.N)

	result7 := new(big.Int)
	result7.Mod(result6, params.N)

	return padToN(result7, params), nil
}
