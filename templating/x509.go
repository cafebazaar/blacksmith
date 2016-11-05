package templating

type KeyCertPair struct {
	Key  string `json:"key"`
	Cert string `json:"cert"`
}

func generateKeyCertPair() (*KeyCertPair, error) {
	// priv, err := GenerateKey(rand.Reader, size)

	keyCertPair := &KeyCertPair{
		Key:  "123",
		Cert: "465",
	}
	return keyCertPair, nil
}
