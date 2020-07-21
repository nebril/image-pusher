main:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -tags containers_image_openpgp -ldflags '-w -extldflags "-static"' -o main main.go

zip: main
	zip pusher.zip policy.json main

clean:
	rm main pusher.zip

all: main zip
