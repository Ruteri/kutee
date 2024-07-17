package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"oram_store/common"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:  "url",
		Value: "http://localhost:8071",
		Usage: "store service url",
	},
}

var keyFlag cli.Flag = &cli.StringFlag{
	Name:     "key",
	Required: true,
	Usage:    "32 bytes key to use, if not 32 bytes it will be hashed instead",
}

var valueFlag cli.Flag = &cli.StringFlag{
	Name:     "value",
	Required: true,
	Usage:    "512 bytes value to use, if not 512 bytes it will be right-padded",
}

func main() {
	app := &cli.App{
		Name:  "oram store client",
		Usage: "Interact with oram store server",
		Flags: flags,
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "get",
				Usage: "Retrieves data from oram map",
				Flags: append([]cli.Flag{
					keyFlag,
				}, flags...),
				Action: runGet,
			},
			&cli.Command{
				Name:  "set",
				Usage: "Inserts data into oram map",
				Flags: append([]cli.Flag{
					keyFlag,
					valueFlag,
				}, flags...),
				Action: runSet,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runGet(cCtx *cli.Context) error {
	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   true,
		JSON:    false,
		Version: common.Version,
	})

	client := &http.Client{}
	key := []byte(cCtx.String("key"))
	if len(key) != 32 {
		key = sha256.New().Sum(key)
	}

	req, err := http.NewRequest("POST", cCtx.String("url")+"/api/get", bytes.NewBuffer(key))
	if err != nil {
		log.Error("could not create request", "err", err)
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Error("could not send request", "err", err)
		return err
	}

	rb, _ := io.ReadAll(res.Body)
	log.With("resp", rb).With("status", res.Status).Info("requested get")

	return nil
}

func runSet(cCtx *cli.Context) error {
	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   true,
		JSON:    false,
		Version: common.Version,
	})

	var reqData [544]byte
	client := &http.Client{}
	key := []byte(cCtx.String("key"))
	if len(key) != 32 {
		key = sha256.New().Sum(key)
	}

	str_value := []byte(cCtx.String("value"))
	if len(str_value) > 512 {
		log.Error("value too long, must be at most 512 bytes")
		return errors.New("value too long")
	}

	copy(reqData[0:32], key)
	copy(reqData[32:544], str_value)

	req, err := http.NewRequest("POST", cCtx.String("url")+"/api/set", bytes.NewBuffer(reqData[:]))
	if err != nil {
		log.Error("could not create request", "err", err)
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Error("could not send request", "err", err)
		return err
	}

	log.With("status", res.Status).Info("requested set")

	return nil
}
