package main

import (
	"context"
	"fmt"

	"github.com/nebril/image-pusher/p"
)

func main() {
	inv := p.Invocation{
		Body: "{ \"Url\": \"https://api.github.com/repos/cilium/cilium/actions/artifacts/14740265/zip\",\"Tag\": \"trolo\" }",
	}

	if err := p.MoveImage(context.TODO(), inv); err != nil {
		fmt.Println(err)
	}
}
