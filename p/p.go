// Package p contains an HTTP Cloud Function.
package p

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

type Invocation struct {
	Body string `json:"body"`
}

func (i Invocation) GetData() InvocationBody {
	body := InvocationBody{}
	json.Unmarshal([]byte(i.Body), &body)

	return body
}

type InvocationBody struct {
	Url string `json:"url"`
	Tag string `json:"tag"`
}

// MoveImage downloads the image from url and pushes it to repository configured by env vars
func MoveImage(ctx context.Context, i Invocation) error {
	data := i.GetData()

	if data.Url == "" {
		return fmt.Errorf("missing Url")
	}
	if data.Tag == "" {
		return fmt.Errorf("missing Tag")
	}

	imageReader, err := downloadImage(data.Url)
	if err != nil {
		return fmt.Errorf("Failed to download image from %s: %s", data.Url, err.Error())
	}

	err = copyImage(imageReader, data.Tag)
	return err
}

func downloadImage(url string) (io.Reader, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not set")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header["Authorization"] = []string{"token " + githubToken}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func copyImage(image io.Reader, tag string) error {
	repository := os.Getenv("TARGET_REPO")
	if repository == "" {
		return fmt.Errorf("TARGET_REPO env var not set")
	}
	username := os.Getenv("TARGET_USER")
	if repository == "" {
		return fmt.Errorf("TARGET_USER env var not set")
	}
	password := os.Getenv("TARGET_PWD")
	if repository == "" {
		return fmt.Errorf("TARGET_PWD env var not set")
	}

	imgBuffer, err := ioutil.ReadAll(image)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("/tmp/image", imgBuffer, 0644)
	if err != nil {
		return err
	}

	imgPath := "/tmp/image"

	files, err := Unzip(imgPath, "/tmp")
	if err == nil && len(files) > 0 {
		imgPath = files[0]
	}
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(files)

	policy, err := signature.NewPolicyFromFile("policy.json")
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}

	destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s:%s", repository, tag))
	if err != nil {
		return err
	}

	srcRef, err := alltransports.ParseImageName("docker-archive:" + imgPath)

	dstDockerAuth := &types.DockerAuthConfig{
		Username: username,
		Password: password,
	}

	sourceCtx := &types.SystemContext{}
	destinationCtx := &types.SystemContext{}
	destinationCtx.DockerAuthConfig = dstDockerAuth

	_, err = copy.Image(context.TODO(), policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures: false,
		SignBy:           "",
		ReportWriter:     os.Stdout,
		SourceCtx:        sourceCtx,
		DestinationCtx:   destinationCtx,
	})

	return err
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
