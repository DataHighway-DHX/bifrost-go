package sr25519

import (
	"encoding/hex"
	"errors"
	"fmt"

	sr25519 "github.com/ChainSafe/go-schnorrkel"
	"github.com/JFJun/bifrost-go/crypto/ss58"
	r255 "github.com/gtank/ristretto255"
)

type KeyPair struct {
	Wif     string
	Address string
}

func GenerateKey() ([]byte, []byte, error) {
	secret, err := sr25519.GenerateMiniSecretKey()
	if err != nil {
		return nil, nil, err
	}
	priv := secret.Encode()
	pub := secret.Public().Encode()
	return priv[:], pub[:], nil
}
func GenerateKeyBySeed(priv []byte) ([]byte, error) {
	var key [32]byte
	copy(key[:], priv[:])
	sk, err := sr25519.NewMiniSecretKeyFromRaw(key)
	if err != nil {
		return nil, err
	}
	pub := sk.Public().Encode()
	return pub[:], nil
}

func CreateAddress(pubKey, prefix []byte) (string, error) {
	return ss58.Encode(pubKey, prefix)
}

func PrivateKeyToAddress(privateKey, prefix []byte) (string, error) {
	var p [32]byte
	copy(p[:], privateKey[:])
	secret, err := sr25519.NewMiniSecretKeyFromRaw(p)
	if err != nil {
		panic(err)
	}

	pub := secret.Public().Encode()
	return ss58.Encode(pub[:], prefix)
}

func PrivateKeyToHex(privateKey []byte) (string, error) {
	if len(privateKey) != 32 {
		return "", errors.New("private key length is not equal 32")
	}
	privHex := hex.EncodeToString(privateKey)
	return "0x" + privHex, nil
}

//
func PrivateKeyToWif(privateKey []byte) (string, error) {
	if len(privateKey) != 32 {
		return "", errors.New("private key length is not equal 32")
	}
	return "", nil
}
func Sign(privateKey, message []byte) ([]byte, error) {
	var sigBytes []byte
	var key, nonce [32]byte
	copy(key[:], privateKey[:32])
	signContext := sr25519.NewSigningContext([]byte("substrate"), message)
	if len(privateKey) == 32 { // Is seed

		sk, err := sr25519.NewMiniSecretKeyFromRaw(key)
		if err != nil {
			return nil, err
		}

		signContext.AppendMessage([]byte("proto-name"), []byte("Schnorr-sig"))
		pub := sk.Public()
		pubc := pub.Encode()
		signContext.AppendMessage([]byte("sign:pk"), pubc[:])

		r, err := sr25519.NewRandomScalar()
		if err != nil {
			return nil, err
		}
		R := r255.NewElement().ScalarBaseMult(r)
		signContext.AppendMessage([]byte("sign:R"), R.Encode([]byte{}))

		sig, err := sr25519.NewSignatureFromHex(R.String())
		if err != nil {
			return nil, err
		}
		sbs := sig.Encode()
		sigBytes = sbs[:]
		verifySigContent := sr25519.NewSigningContext([]byte("substrate"), message)
		ok, err := sk.Public().Verify(sig, verifySigContent)
		if !ok {
			return nil, errors.New("verify sign error")
		}
	} else if len(privateKey) == 64 { //Is private key
		copy(nonce[:], privateKey[32:])
		sk := sr25519.NewSecretKey(key, nonce)
		sig, err := sk.Sign(signContext)
		if err != nil {
			return nil, fmt.Errorf("sr25519 sign error,err=%v", err)
		}
		sbs := sig.Encode()
		sigBytes = sbs[:]
		pub, _ := sk.Public()
		ok, err := pub.Verify(sig, sr25519.NewSigningContext([]byte("substrate"), message))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("verify sign error")
		}
	}
	return sigBytes[:], nil

}
