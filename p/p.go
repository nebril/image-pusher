// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

// MoveImage downloads the image from url and pushes it to repository configured by env vars
func MoveImage(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Url string `json:"url"`
		Tag string `json:"tag"`
	}

	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - malformed json"))
		return
	}
	if d.Url == "" || d.Tag == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - missing arguments"))
		return
	}

	imageReader, err := downloadImage(d.Url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to download image from %s: %s", d.Url, err.Error())
	}

	err = copyImage(imageReader, d.Tag)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to copy image: %s", err.Error())
	}
}

func downloadImage(url string) (io.Reader, error) {
	resp, err := http.Get(url)
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

	policy, err := signature.DefaultPolicy(nil)
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

	srcRef, err := alltransports.ParseImageName("docker-archive:image")

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
