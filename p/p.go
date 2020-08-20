// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

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
	resp, err := http.Get(url)
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

	srcRef, err := alltransports.ParseImageName("docker-archive:/tmp/image")

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
