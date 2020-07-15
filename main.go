package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nebril/image-pusher/p"
)

func main() {
	lambda.Start(p.MoveImage)
}
