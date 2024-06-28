package httpserver

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type KuteeAPI struct {
	AuthenticatedUsers []BasicAuth
	PasswordHasher     func(string) []byte
}

func NewKuteeAPI(authorizedUsers map[string][]byte, pwHasher func(string) []byte) *KuteeAPI {
	api := &KuteeAPI{
		AuthenticatedUsers: make([]BasicAuth, 0, len(authorizedUsers)),
		PasswordHasher:     pwHasher,
	}
	for u, ph := range authorizedUsers {
		api.AuthenticatedUsers = append(api.AuthenticatedUsers, BasicAuth{u, ph})
	}
	return api
}

type BasicAuth struct {
	Username     string
	PasswordHash []byte
}

func (s *KuteeAPI) AuthenticateAndHandle(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "missing authentication", http.StatusUnauthorized)
			return
		}

		ph := s.PasswordHasher(p)

		for _, authenticatedUser := range s.AuthenticatedUsers {
			if authenticatedUser.Username == u {
				if bytes.Equal(authenticatedUser.PasswordHash, ph) {
					handler(w, r)
					return
				}
			}
		}

		http.Error(w, "", http.StatusUnauthorized)
	}
}

const MaxImageSize = 1024 * 1024 * 500 // 500MiB
func (s *KuteeAPI) uploadImageTarball(w http.ResponseWriter, r *http.Request) {
	// Adjusted from https://github.com/Freshman-tech/file-upload/commit/f1638a7d39057122f97dd015bb1f5f3cda196ac0 (MIT)
	r.Body = http.MaxBytesReader(w, r.Body, MaxImageSize)
	if err := r.ParseMultipartForm(MaxImageSize); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// The argument to FormFile must match the name attribute
	// of the file input on the frontend
	file, fileHeader, err := r.FormFile("image-tarball")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if filepath.Ext(fileHeader.Filename) != ".tar" {
		http.Error(w, "only .tar images are supported", http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
	imgDir := os.TempDir() + "/image"
	err = os.MkdirAll(imgDir, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new file in the uploads directory
	imagePath := imgDir + "/" + fileHeader.Filename
	dst, err := os.Create(imagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	// at the specified destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("minikube", "image", "load", imagePath)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *KuteeAPI) startWorkload(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("minikube", "kubectl", "--", "apply", "-f", "workload.yaml")
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
