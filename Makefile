.PHONY: lifx
lifx:
	GOARCH=amd64 GOOS=linux go build -o lifx cmd/lifx/main.go
	scp lifx adam@100.91.70.121:~/
	rsync -aHv --stats --progress /Users/adam/Scripts/apps/lifx/curves/ adam@100.91.70.121:/home/adam/curves/
