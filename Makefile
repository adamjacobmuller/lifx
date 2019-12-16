.PHONY: lifx
lifx:
	GOOS=linux go build -o lifx cmd/lifx/main.go
	scp lifx adam@10.0.8.3:~/
	rsync -aHv --stats --progress /Users/adam/Scripts/apps/lifx/curves/ adam@10.0.8.3:/home/adam/curves/
	ssh adam@10.0.8.3 /usr/bin/sudo /bin/systemctl restart lifx
