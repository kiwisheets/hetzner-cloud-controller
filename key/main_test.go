package key

import (
	"fmt"
	"testing"
)

func TestGenerateSSHkeyPair(t *testing.T) {
	priv, pub, err := GenerateSSHkeyPair()

	if err != nil {
		t.Log("Failed to generate SSH keypair")
		t.Error(err)
	}

	fmt.Println(priv)
	fmt.Println(pub)
}
