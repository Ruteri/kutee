package main

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/flashbots/go-template/common"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags []cli.Flag = []cli.Flag{
	&cli.BoolFlag{
		Name:  "log-json",
		Value: false,
		Usage: "log in JSON format",
	},
	&cli.BoolFlag{
		Name:  "log-debug",
		Value: false,
		Usage: "log debug messages",
	},
	&cli.StringFlag{
		Name:  "username",
		Value: "test",
		Usage: "username to authenticate with",
	},
	&cli.StringFlag{
		Name:  "password",
		Value: "test",
		Usage: "password to authenticate with",
	},
	&cli.StringFlag{
		Name:  "url",
		Value: "http://localhost:8087",
		Usage: "kutee service url",
	},
}

var imageFlag cli.Flag = &cli.StringFlag{
	Name:  "image",
	Value: "img.tar",
	Usage: "path to image to upload",
}

func main() {
	app := &cli.App{
		Name:   "httpserver",
		Usage:  "Serve API, and metrics",
		Flags:  flags,
		Action: runCli,
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "upload",
				Usage: "Uploads an image tarball",
				Flags: append([]cli.Flag{
					imageFlag,
				}, flags...),
				Action: runUpload,
			},
			&cli.Command{
				Name:   "start",
				Usage:  "Requests workload start",
				Flags:  flags,
				Action: runStart,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runUpload(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	errCh := make(chan error, 1)
	go func() {
		defer w.Close()
		defer m.Close()
		part, err := m.CreateFormFile("image-tarball", cCtx.String("image"))
		if err != nil {
			log.Error("could not create multipart reader from file", "err", err)
			errCh <- err
			return
		}
		file, err := os.Open(cCtx.String("image"))
		if err != nil {
			log.Error("could not open the file", "err", err)
			errCh <- err
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			log.Error("could not copy part", "err", err)
			errCh <- err
			return
		}
		errCh <- nil
	}()

	errCh2 := make(chan error, 1)
	go func() {
		client := &http.Client{}
		req, err := http.NewRequest("POST", cCtx.String("url")+"/api/upload_image", r)
		if err != nil {
			log.Error("could not create request", "err", err)
			errCh2 <- err
			return
		}
		req.Header.Add("Content-Type", m.FormDataContentType())
		req.SetBasicAuth(cCtx.String("username"), cCtx.String("password"))

		res, err := client.Do(req)
		if err != nil {
			log.Error("could not upload image", "err", err)
			errCh2 <- err
			return
		}

		rb, _ := io.ReadAll(res.Body)
		log.With("resp", rb).With("status", res.Status).Info("requested upload")

		errCh2 <- nil
	}()

	mpErr := <-errCh
	postErr := <-errCh2

	if postErr != nil {
		log.Error("could not post", "err", postErr)
	}

	if mpErr != nil {
		log.Error("could upload multipart", "err", mpErr)
	}

	if postErr != nil || mpErr != nil {
		return errors.New("upload failed")
	}

	return nil
}

func runStart(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	client := &http.Client{}
	req, err := http.NewRequest("GET", cCtx.String("url")+"/api/start_workload", nil)
	if err != nil {
		log.Error("could not create request", "err", err)
		return err
	}
	req.SetBasicAuth(cCtx.String("username"), cCtx.String("password"))

	res, err := client.Do(req)
	if err != nil {
		log.Error("could not send request", "err", err)
		return err
	}

	rb, _ := io.ReadAll(res.Body)
	log.With("resp", rb).With("status", res.Status).Info("requested start")

	return nil
}

func runCli(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	log.Info("Starting the project")

	log.Debug("debug message")
	log.Info("info message")
	log.With("key", "value").Warn("warn message")
	log.Error("error message", "err", errors.ErrUnsupported)
	return nil
}
