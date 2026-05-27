package googleworkspace

// Store reads and writes per-service token blobs in the OS credential store.
type Store struct {
	Service string
	Account string
}

func NewStore(svc Service) Store {
	account := svc.KeychainAccount()
	if a := overrideAccount(svc.ID); a != "" {
		account = a
	}
	return Store{
		Service: KeychainServiceNameForStore(),
		Account: account,
	}
}

func overrideAccount(serviceID string) string {
	// GOOGLE_GMAIL_KEYCHAIN_ACCOUNT still supported
	return getenvService(serviceID, "KEYCHAIN_ACCOUNT")
}
