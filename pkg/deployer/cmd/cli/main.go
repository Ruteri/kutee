package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"kutee/common"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags []cli.Flag = []cli.Flag{
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
		Usage: "deployer service url",
	},
}

var deploymentFileFlag cli.Flag = &cli.StringFlag{
	Name:  "deployment-file",
	Value: "./deployment.yaml",
	Usage: "path to deployment file",
}

var tmpBundleDirFlag cli.Flag = &cli.StringFlag{
	Name:  "bundle-dir",
	Value: "./bundle",
	Usage: "path to directory to keep the bundle in",
}

func main() {
	app := &cli.App{
		Name:  "Deployer cli",
		Usage: "deploys your app to Tstack",
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "deploy",
				Usage: "Deploys an application to Tstack",
				Flags: append([]cli.Flag{
					deploymentFileFlag,
					tmpBundleDirFlag,
				}, flags...),
				Action: runDeploy,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runDeploy(cCtx *cli.Context) error {
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")

	log := common.SetupLogger(&common.LoggingOpts{
		Debug:   logDebug,
		JSON:    logJSON,
		Version: common.Version,
	})

	// 1. Fetch all images from the deployment file
	deploymentFileContent, err := os.ReadFile(cCtx.String("deployment-file"))
	if err != nil {
		panic(err)
	}

	images := []string{}
	for _, line := range strings.Split(string(deploymentFileContent), "\n") {
		line := strings.TrimSpace(line)
		if len(line) > 7 && line[0:7] == "image: " {
			images = append(images, strings.Split(line, " ")[1])
		}
	}

	// 2. Copy the deployment file to the bundle directory
	bundle_dir := cCtx.String("bundle-dir")
	err = exec.Command("mkdir", "-p", bundle_dir).Run()
	if err != nil {
		panic(err)
	}

	err = exec.Command("cp", cCtx.String("deployment-file"), bundle_dir+"/deployment.yaml").Run()
	if err != nil {
		panic(err)
	}

	// 3. Export all images to the bundle directory
	image_archives := []string{}
	for _, image := range images {
		colon_escaped_image := strings.ReplaceAll(image, ":", "-")
		image_tar_file := bundle_dir + "/" + colon_escaped_image + ".tar"
		// TODO: replace also in the deployment file!

		output, err := exec.Command("docker", "image", "save", image, "-o", image_tar_file).CombinedOutput()
		if err != nil {
			fmt.Println(exec.Command("docker", "image", "save", image, "-o", image_tar_file).String())
			fmt.Println(string(output))
			panic(err)
		}
		image_archives = append(image_archives, image_tar_file)

		err = exec.Command("sed", "-i", "-e", "s/"+image+"/"+colon_escaped_image+"/", bundle_dir+"/deployment.yaml").Run()
		if err != nil {
			panic(err)
		}
	}

	// 4. Tar the bundle and upload to Tstack server
	files_to_archive := []string{bundle_dir + "/deployment.yaml"}
	files_to_archive = append(files_to_archive, image_archives...)

	tar_args := []string{"--create", "-f", "bundle.tar", "-z"}
	tar_args = append(tar_args, files_to_archive...)

	err = exec.Command("tar", tar_args...).Run()
	if err != nil {
		panic(err)
	}

	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	errCh := make(chan error, 1)
	go func() {
		defer w.Close()
		defer m.Close()
		part, err := m.CreateFormFile("deployment-bundle", "bundle.tar")
		if err != nil {
			log.Error("could not create multipart reader from file", "err", err)
			errCh <- err
			return
		}
		file, err := os.Open("bundle.tar")
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
		req, err := http.NewRequest("POST", cCtx.String("url")+"/api/deploy", r)
		if err != nil {
			log.Error("could not create request", "err", err)
			errCh2 <- err
			return
		}
		req.Header.Add("Content-Type", m.FormDataContentType())
		req.SetBasicAuth(cCtx.String("username"), cCtx.String("password"))

		res, err := client.Do(req)
		if err != nil {
			log.Error("could not upload bundle", "err", err)
			errCh2 <- err
			return
		}

		log.With("status", res.Status).Info("requested upload")

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
