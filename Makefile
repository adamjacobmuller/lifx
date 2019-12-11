.PHONY: lifx
lifx:
	GOOS=linux go build -o lifx cmd/lifx/main.go
	scp lifx adam@10.0.8.3:~/
	ssh adam@10.0.8.3 /usr/bin/sudo /bin/systemctl restart lifx
