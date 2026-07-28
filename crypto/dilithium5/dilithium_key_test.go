package dilithium5_test

import (
	"testing"

	bip39 "github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"zytherion/crypto/dilithium5"
)

// ── Key generation tests ──────────────────────────────────────────────────────

func TestGenPrivKey(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)
	require.NotNil(t, sk)
	require.Equal(t, dilithium5.PrivKeySize, len(sk.Bytes()))
	require.Equal(t, "dilithium5", sk.Type())
}

func TestPubKeyDerivation(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)

	pk := sk.PubKey()
	require.NotNil(t, pk)
	require.Equal(t, dilithium5.PubKeySize, len(pk.Bytes()))
	require.Equal(t, "dilithium5", pk.Type())
}

func TestAddress(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)
	pk := sk.PubKey()

	addr := pk.Address()
	require.NotNil(t, addr)
	require.Equal(t, 20, len(addr), "Cosmos address must be 20 bytes")
}

// ── Sign + Verify tests ───────────────────────────────────────────────────────

func TestSignAndVerify(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)
	pk := sk.PubKey()

	msg := []byte("Zytherion quantum dollar coin")
	sig, err := sk.Sign(msg)
	require.NoError(t, err)
	require.Equal(t, dilithium5.SigSize, len(sig), "Dilithium5 signature must be 4595 bytes")

	require.True(t, pk.VerifySignature(msg, sig), "valid sig must verify")
}

func TestVerifyRejectsTamperedMessage(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)
	pk := sk.PubKey()

	msg := []byte("original message")
	sig, err := sk.Sign(msg)
	require.NoError(t, err)

	tampered := []byte("tampered message!!")
	require.False(t, pk.VerifySignature(tampered, sig), "tampered message must not verify")
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)
	pk := sk.PubKey()

	msg := []byte("message")
	sig, err := sk.Sign(msg)
	require.NoError(t, err)

	// flip one byte in the middle of the signature
	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[len(tampered)/2] ^= 0xFF

	require.False(t, pk.VerifySignature(msg, tampered), "tampered sig must not verify")
}

func TestVerifyRejectsCrossKeySignature(t *testing.T) {
	sk1, _ := dilithium5.GenPrivKey()
	sk2, _ := dilithium5.GenPrivKey()

	msg := []byte("cross key test")
	sig, err := sk1.Sign(msg)
	require.NoError(t, err)

	// sk2's pubkey must not accept a sig from sk1
	require.False(t, sk2.PubKey().VerifySignature(msg, sig), "cross-key sig must not verify")
}

// ── Determinism tests ─────────────────────────────────────────────────────────

func TestDeterministicSigning(t *testing.T) {
	sk, err := dilithium5.GenPrivKey()
	require.NoError(t, err)

	msg := []byte("deterministic")
	sig1, err := sk.Sign(msg)
	require.NoError(t, err)
	sig2, err := sk.Sign(msg)
	require.NoError(t, err)

	require.Equal(t, sig1, sig2, "Dilithium5 signing must be deterministic (required by Green-BFT)")
}

// ── Mnemonic derivation tests ─────────────────────────────────────────────────

func TestMnemonicDerive_Deterministic(t *testing.T) {
	entropy, err := bip39.NewEntropy(256)
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)

	algo := dilithium5.Dilithium5Algo
	derive := algo.Derive()
	generate := algo.Generate()

	// Derive twice from same mnemonic + path → must produce identical raw bytes
	bz1, err := derive(mnemonic, "", "m/44'/118'/0'/0/0")
	require.NoError(t, err)
	bz2, err := derive(mnemonic, "", "m/44'/118'/0'/0/0")
	require.NoError(t, err)
	require.Equal(t, bz1, bz2, "same mnemonic + path must produce same entropy")

	// And the same entropy must produce the same keypair
	sk1 := generate(bz1)
	sk2 := generate(bz2)
	require.Equal(t, sk1.Bytes(), sk2.Bytes(), "same entropy must produce same private key")
}

func TestMnemonicDerive_DifferentPaths(t *testing.T) {
	entropy, err := bip39.NewEntropy(256)
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)

	algo := dilithium5.Dilithium5Algo
	derive := algo.Derive()

	bz0, _ := derive(mnemonic, "", "m/44'/118'/0'/0/0")
	bz1, _ := derive(mnemonic, "", "m/44'/118'/0'/0/1")

	require.NotEqual(t, bz0, bz1, "different HD paths must produce different keys")
}

func TestMnemonicDerive_DifferentMnemonics(t *testing.T) {
	e1, err := bip39.NewEntropy(256)
	require.NoError(t, err)
	m1, err := bip39.NewMnemonic(e1)
	require.NoError(t, err)
	e2, err := bip39.NewEntropy(256)
	require.NoError(t, err)
	m2, err := bip39.NewMnemonic(e2)
	require.NoError(t, err)

	algo := dilithium5.Dilithium5Algo
	derive := algo.Derive()

	bz1, _ := derive(m1, "", "m/44'/118'/0'/0/0")
	bz2, _ := derive(m2, "", "m/44'/118'/0'/0/0")

	require.NotEqual(t, bz1, bz2, "different mnemonics must produce different keys")
}

func TestMnemonicDerive_InvalidMnemonic(t *testing.T) {
	algo := dilithium5.Dilithium5Algo
	derive := algo.Derive()

	_, err := derive("this is not a valid mnemonic at all", "", "m/44'/118'/0'/0/0")
	require.Error(t, err, "invalid mnemonic must return error")
}

// ── Marshal / Unmarshal tests ─────────────────────────────────────────────────

func TestPubKeyMarshalRoundtrip(t *testing.T) {
	sk, _ := dilithium5.GenPrivKey()
	pk := sk.PubKey().(*dilithium5.PubKey)

	bz, err := pk.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	pk2 := &dilithium5.PubKey{}
	err = pk2.Unmarshal(bz)
	require.NoError(t, err)
	require.True(t, pk.Equals(pk2), "pubkey marshal roundtrip must be lossless")
}

func TestPrivKeyMarshalRoundtrip(t *testing.T) {
	sk, _ := dilithium5.GenPrivKey()

	bz, err := sk.Marshal()
	require.NoError(t, err)

	sk2 := &dilithium5.PrivKey{}
	err = sk2.Unmarshal(bz)
	require.NoError(t, err)
	require.True(t, sk.Equals(sk2), "privkey marshal roundtrip must be lossless")
}

// ── Equality tests ────────────────────────────────────────────────────────────

func TestEquality(t *testing.T) {
	sk1, _ := dilithium5.GenPrivKey()
	sk2, _ := dilithium5.GenPrivKey()

	require.True(t, sk1.Equals(sk1), "key equals itself")
	require.False(t, sk1.Equals(sk2), "different keys are not equal")
	require.True(t, sk1.PubKey().Equals(sk1.PubKey()), "pubkey equals itself")
	require.False(t, sk1.PubKey().Equals(sk2.PubKey()), "different pubkeys are not equal")
}

func TestKeyringOption(t *testing.T) {
	// Import necessary SDK crypto/keyring packages.
	// Since we are in dilithium5_test package, we can test using keyring.NewInMemory and passing the custom Option.
	ir := codectypes.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(ir)

	// Custom keyring option matching cmd.dilithium5KeyringOption
	opt := keyring.Option(func(options *keyring.Options) {
		options.SupportedAlgos = keyring.SigningAlgoList{
			hd.Secp256k1,
			dilithium5.Dilithium5Algo,
		}
	})

	kb := keyring.NewInMemory(codec, opt)
	algos, _ := kb.SupportedAlgorithms()

	// Verify dilithium5 is registered in SupportedAlgos list
	found := false
	for _, algo := range algos {
		if algo.Name() == "dilithium5" {
			found = true
			break
		}
	}
	require.True(t, found, "dilithium5 algorithm must be supported by the keyring")
}
