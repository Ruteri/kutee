package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"simple-key-service/httpserver"

	"simple-key-service/common"

	"github.com/ethereum/go-ethereum/common/hexutil"
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
		Name:  "url",
		Value: "http://localhost:8088",
		Usage: "key service url",
	},
}

var serviceFlag cli.Flag = &cli.StringFlag{
	Name:  "service",
	Value: "default",
	Usage: "name of the service",
}
var tokenFlag cli.Flag = &cli.StringFlag{
	Name:  "token",
	Value: "0x0000000000000000000000000000000000000000000000000000000000000000",
	Usage: "hex-encoded 32byte random",
}

var plaintextFlag cli.Flag = &cli.StringFlag{
	Name:  "plaintext",
	Value: "0xffffffff",
	Usage: "hex-encoded string to encrypt",
}

var ciphertextFlag cli.Flag = &cli.StringFlag{
	Name:  "ciphertext",
	Value: "0xffffffff",
	Usage: "hex-encoded string to decrypt",
}

func main() {
	app := &cli.App{
		Name:   "httpserver",
		Usage:  "Serve API, and metrics",
		Flags:  flags,
		Action: runCli,
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "derive-pub",
				Usage: "Calls derive pub",
				Flags: append([]cli.Flag{
					serviceFlag, tokenFlag,
				}, flags...),
				Action: runDerive,
			},
			&cli.Command{
				Name:  "get-pub",
				Usage: "Calls get pub",
				Flags: append([]cli.Flag{
					serviceFlag,
				}, flags...),
				Action: runGetPub,
			},
			&cli.Command{
				Name:  "encrypt",
				Usage: "Calls encrypt",
				Flags: append([]cli.Flag{
					serviceFlag, plaintextFlag,
				}, flags...),
				Action: runEncrypt,
			},
			&cli.Command{
				Name:  "decrypt",
				Usage: "Calls decrypt",
				Flags: append([]cli.Flag{
					tokenFlag, ciphertextFlag,
				}, flags...),
				Action: runDecrypt,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func postJSON(url string, body any) ([]byte, error) {
	reqJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func runDerive(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	res, err := postJSON(cCtx.String("url")+"/api/derive_pubkey", httpserver.DerivePubkeyRequest{
		ServiceName: cCtx.String("service"),
		RandomToken: [32]byte(hexutil.MustDecode(cCtx.String("token"))),
	})
	if err != nil {
		log.Error("could not request", "err", err)
		return err
	}

	fmt.Println(string(res))

	return nil
}

func runGetPub(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	res, err := postJSON(cCtx.String("url")+"/api/get_pubkey", httpserver.GetPubkeyRequest{
		ServiceName: cCtx.String("service"),
	})
	if err != nil {
		log.Error("could not request", "err", err)
		return err
	}

	fmt.Println(string(res))

	return nil
}

func runEncrypt(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	res, err := postJSON(cCtx.String("url")+"/api/encrypt", httpserver.EncryptRequest{
		ServiceName: cCtx.String("service"),
		Plaintext:   hexutil.MustDecode(cCtx.String("plaintext")),
	})
	if err != nil {
		log.Error("could not request", "err", err)
		return err
	}

	result := httpserver.EncryptResponse{}
	json.Unmarshal(res, &result)

	fmt.Println(string(res), hexutil.Encode(result.Ciphertext))

	return nil
}

func runDecrypt(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	res, err := postJSON(cCtx.String("url")+"/api/decrypt", httpserver.DecryptRequest{
		RandomToken: [32]byte(hexutil.MustDecode(cCtx.String("token"))),
		Ciphertext:  hexutil.MustDecode(cCtx.String("ciphertext")),
	})
	if err != nil {
		log.Error("could not request", "err", err)
		return err
	}

	result := httpserver.DecryptResponse{}
	json.Unmarshal(res, &result)

	fmt.Println(string(res), string(result.Plaintext))

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
