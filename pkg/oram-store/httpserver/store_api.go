package httpserver

import (
	"net/http"
	"unsafe"

	"oram_store/odsl"
)

type StoreAPI struct {
	// TODO: should this be per-service, with a secret separating each service?
	omap odsl.OMapBinding
}

func NewKuteeAPI() *StoreAPI {
	api := &StoreAPI{
		omap: odsl.NewOMapBinding(),
	}
	api.omap.InitEmpty(1000)
	return api
}

type DerivePubkeyRequest struct {
	ServiceName string   `json:"service_name"`
	RandomToken [32]byte `json:"random_token"`
}

type DerivePubkeyResponse struct {
	DerivedPubkey []byte `json:"derived_pubkey"`
}

func (s *StoreAPI) set(w http.ResponseWriter, r *http.Request) {
	var req []byte = make([]byte, 512+32)
	n, err := r.Body.Read(req) // should be max bytes reader I guess
	if err != nil && err.Error() != "EOF" {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if n != 512+32 {
		http.Error(w, "invalid body length, expected 544", http.StatusBadRequest)
		return
	}

	var key [32]byte
	var val [512]byte

	copy(key[:], req[0:32])
	copy(val[:], req[32:544])

	s.omap.Insert(getUintPtr(&key), getUintPtr(&val))

	w.Write(nil)
}

func (s *StoreAPI) get(w http.ResponseWriter, r *http.Request) {
	var req []byte = make([]byte, 32)
	n, err := r.Body.Read(req) // should be max bytes reader I guess
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if n != 32 {
		http.Error(w, "invalid body length, expected 32", http.StatusBadRequest)
		return
	}

	var key [32]byte
	copy(key[:], req[0:32])

	var findRes [512]byte
	foundFlag := s.omap.Find(getUintPtr(&key), getUintPtr(&findRes))

	// TODO: Can we respond differently based on foundFlag without leaking info?
	if foundFlag {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(findRes[:])
		return
	}

	http.Error(w, "not found", http.StatusBadRequest)
	return
}

func getUintPtr[T any](num *T) uintptr {
	return uintptr(unsafe.Pointer(num))
}
