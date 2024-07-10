package httpserver

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/tyler-smith/go-bip32"
)

type KeyServiceAPI struct {
	localSecret    []byte
	lock           sync.Mutex
	derivedPubkeys map[string][]byte
}

func NewKeyServiceAPI() *KeyServiceAPI {
	seed, err := bip32.NewSeed()
	if err != nil {
		panic(err)
	}

	return &KeyServiceAPI{
		localSecret:    seed,
		lock:           sync.Mutex{},
		derivedPubkeys: make(map[string][]byte),
	}
}

func NewKeyServiceAPIFromSeed(seed []byte) *KeyServiceAPI {
	return &KeyServiceAPI{
		localSecret:    seed,
		lock:           sync.Mutex{},
		derivedPubkeys: make(map[string][]byte),
	}
}

type DerivePubkeyRequest struct {
	ServiceName string   `json:"service_name"`
	RandomToken [32]byte `json:"random_token"`
}

type DerivePubkeyResponse struct {
	DerivedPubkey []byte `json:"derived_pubkey"`
}

func (s *KeyServiceAPI) handleDerivePubkey(w http.ResponseWriter, r *http.Request) {
	var req DerivePubkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	if _, found := s.derivedPubkeys[req.ServiceName]; found {
		http.Error(w, "already registered", http.StatusBadRequest)
		return
	}

	key, err := bip32.NewMasterKey(append(s.localSecret, req.RandomToken[:]...))
	if err != nil {
		http.Error(w, "could not recover master key", http.StatusInternalServerError)
		return
	}

	ecdaPrivateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		http.Error(w, "could not load ecdsa key", http.StatusInternalServerError)
		return
	}

	ecdaPublicKey := ecdaPrivateKey.Public().(*ecdsa.PublicKey)
	derivedPubkey := crypto.CompressPubkey(ecdaPublicKey)

	s.derivedPubkeys[req.ServiceName] = derivedPubkey

	// Trigger deferred unlock before serializing
	defer func(w http.ResponseWriter, derivedPubkey []byte) {
		// Prepare the response
		res := DerivePubkeyResponse{
			DerivedPubkey: derivedPubkey,
		}

		// Encode the response struct into JSON format
		responseJSON, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set Content-Type header and write the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(responseJSON)
		if err != nil {
			// Handle error while writing response
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}(w, derivedPubkey)
}

type GetPubkeyRequest struct {
	ServiceName string `json:"service_name"`
}

type GetPubkeyResponse struct {
	DerivedPubkey []byte `json:"derived_pubkey"`
}

func (s *KeyServiceAPI) handleGetPubkey(w http.ResponseWriter, r *http.Request) {
	var req GetPubkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	derivedPubkey, found := s.derivedPubkeys[req.ServiceName]
	if !found {
		http.Error(w, "service not registered", http.StatusBadRequest)
		return
	}

	// Trigger deferred unlock before serializing
	defer func(w http.ResponseWriter, derivedPubkey []byte) {
		// Prepare the response
		res := GetPubkeyResponse{
			DerivedPubkey: derivedPubkey,
		}

		// Encode the response struct into JSON format
		responseJSON, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set Content-Type header and write the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(responseJSON)
		if err != nil {
			// Handle error while writing response
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}(w, derivedPubkey)
}

type EncryptRequest struct {
	ServiceName string `json:"service_name"`
	Plaintext   []byte `json:"plaintext"` // assuming https, otherwise should be encrypted to the key service
}

type EncryptResponse struct {
	Ciphertext []byte `json:"ciphertext"`
}

func (s *KeyServiceAPI) handleEncrypt(w http.ResponseWriter, r *http.Request) {
	var req EncryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.lock.Lock()
	derivedPubkey, found := s.derivedPubkeys[req.ServiceName]
	if !found {
		s.lock.Unlock()
		http.Error(w, "service not registered", http.StatusBadRequest)
		return
	}
	s.lock.Unlock()

	ecdsaPubkey, err := crypto.DecompressPubkey(derivedPubkey)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "could not parse derived key", http.StatusInternalServerError)
		return
	}

	eciesPubkey := ecies.ImportECDSAPublic(ecdsaPubkey)
	ct, err := ecies.Encrypt(rand.Reader, eciesPubkey, req.Plaintext, nil, nil)
	if err != nil {
		http.Error(w, "could not encrypt", http.StatusInternalServerError)
		return
	}

	// Prepare the response
	res := EncryptResponse{
		Ciphertext: ct,
	}

	// Encode the response struct into JSON format
	responseJSON, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set Content-Type header and write the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(responseJSON)
	if err != nil {
		// Handle error while writing response
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type DecryptRequest struct {
	RandomToken [32]byte `json:"random_token"`
	Ciphertext  []byte   `json:"ciphertext"`
}

type DecryptResponse struct {
	Plaintext []byte `json:"plaintext"` // assuming https, otherwise should be encrypted to the requesting service
}

func (s *KeyServiceAPI) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	key, err := bip32.NewMasterKey(append(s.localSecret, req.RandomToken[:]...))
	if err != nil {
		http.Error(w, "could not recover master key", http.StatusInternalServerError)
		return
	}

	ecdsaKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		http.Error(w, "could not load ecdsa key", http.StatusInternalServerError)
		return
	}

	eciesKey := ecies.ImportECDSA(ecdsaKey)
	pt, err := eciesKey.Decrypt(req.Ciphertext, nil, nil)
	if err != nil {
		http.Error(w, "could not decrypt", http.StatusInternalServerError)
		return
	}

	// Prepare the response
	res := DecryptResponse{
		Plaintext: pt,
	}

	// Encode the response struct into JSON format
	responseJSON, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set Content-Type header and write the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(responseJSON)
	if err != nil {
		// Handle error while writing response
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
