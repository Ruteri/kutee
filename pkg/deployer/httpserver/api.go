package httpserver

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type DeployerAPI struct {
	BaseImagePath   string
	RunTdScriptPath string

	AuthenticatedUsers []BasicAuth
	PasswordHasher     func(string) []byte

	log *slog.Logger
}

func NewDeployerAPI(baseImagePath string, runTdScriptPath string, authorizedUsers map[string][]byte, pwHasher func(string) []byte, log *slog.Logger) *DeployerAPI {
	api := &DeployerAPI{
		BaseImagePath:      baseImagePath,
		RunTdScriptPath:    runTdScriptPath,
		AuthenticatedUsers: make([]BasicAuth, 0, len(authorizedUsers)),
		PasswordHasher:     pwHasher,
		log:                log,
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

func (s *DeployerAPI) AuthenticateAndHandle(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
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
func (s *DeployerAPI) deploy(w http.ResponseWriter, r *http.Request) {
	// Adjusted from https://github.com/Freshman-tech/file-upload/commit/f1638a7d39057122f97dd015bb1f5f3cda196ac0 (MIT)
	r.Body = http.MaxBytesReader(w, r.Body, MaxImageSize)
	if err := r.ParseMultipartForm(MaxImageSize); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// The argument to FormFile must match the name attribute
	// of the file input on the frontend
	file, fileHeader, err := r.FormFile("deployment-bundle")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if filepath.Ext(fileHeader.Filename) != ".tar" {
		http.Error(w, "only .tar archives are supported", http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
	tmpDir := os.TempDir()
	err = os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		s.log.Error("could not create temp dir", "err", err)
		http.Error(w, "could not create temp dir", http.StatusInternalServerError)
		return
	}
	// TODO: cleanup the temp dir!

	fmt.Println(tmpDir)

	// Create a new file in the uploads directory
	bundlePath := tmpDir + "/bundle.tar"
	dst, err := os.Create(bundlePath)
	if err != nil {
		s.log.Error("could not create bundle file", "err", err)
		http.Error(w, "could not create bundle file", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	// at the specified destination
	_, err = io.Copy(dst, file)
	if err != nil {
		s.log.Error("could not save bundle file", "err", err)
		http.Error(w, "could not save bundle file", http.StatusInternalServerError)
		return
	}

	// 1. Unpack the bundle archive
	err = exec.Command("tar", "-x", "-z", "-f", bundlePath, "-C", tmpDir).Run()
	if err != nil {
		s.log.Error("could not unpack bundle", "err", err)
		http.Error(w, "could not unpack bundle", http.StatusInternalServerError)
		return
	}

	// 2. Make sure the archive contains the kubernetes deployment.yaml and container images

	// 3. Copy the base VM image
	vmImage := tmpDir + "/image.qcow2"

	err = exec.Command("cp", s.BaseImagePath, vmImage).Run()
	if err != nil {
		s.log.Error("could not copy the baseimage", "err", err)
		http.Error(w, "could not copy the baseimage", http.StatusInternalServerError)
		return
	}

	// 4. Install the unpacked files into the image
	cmd := exec.Command("find", tmpDir+"/bundle/", "-type", "f", "-exec", "sh", "-c", "sudo virt-customize -a "+vmImage+" --copy-in {}:/kutee/", ";")
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.log.With("cmd", cmd.String()).With("output", output).Error("could not load images into the baseimage", "err", err)
		http.Error(w, "could not load images into the baseimage", http.StatusInternalServerError)
		return
	}
	s.log.With("cmd", cmd.String()).With("output", string(output)).Info("installed files into the image")

	// 5. Take the measurement of the image

	// 6. Start the VM
	cmd = exec.Command("bash", s.RunTdScriptPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TD_IMG="+vmImage)
	output, err = cmd.CombinedOutput()
	if err != nil {
		s.log.With("output", output).Error("could not run the image", "err", err)
		http.Error(w, "could not run the image", http.StatusInternalServerError)
		return
	}
	s.log.With("output", output).Info("Running TD")

	// 7. Return the measurement

	w.WriteHeader(http.StatusOK)
}
