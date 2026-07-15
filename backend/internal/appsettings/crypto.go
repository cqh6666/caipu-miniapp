package appsettings

import "github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"

type cipherBox struct {
	box *credentialcipher.Box
}

func newCipherBox(secret string) *cipherBox {
	box, err := credentialcipher.New(credentialcipher.Key{Version: "v1", Secret: secret}, nil)
	if err != nil {
		panic(err)
	}
	return &cipherBox{box: box}
}

func newVersionedCipherBox(secret, version string, previous []credentialcipher.Key) (*cipherBox, error) {
	box, err := credentialcipher.New(credentialcipher.Key{Version: version, Secret: secret}, previous)
	if err != nil {
		return nil, err
	}
	return &cipherBox{box: box}, nil
}

func (b *cipherBox) Encrypt(plain string) (string, error) {
	return b.box.Encrypt(plain)
}

func (b *cipherBox) Decrypt(ciphertext string) (string, error) {
	return b.box.Decrypt(ciphertext)
}
